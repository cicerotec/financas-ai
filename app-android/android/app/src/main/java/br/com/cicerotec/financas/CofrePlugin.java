package br.com.cicerotec.financas;

import android.content.Context;
import android.content.SharedPreferences;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.nio.charset.StandardCharsets;
import java.security.KeyStore;

import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;

/**
 * Cofre: o armazenamento do login do app (item 54, fase 2; regra R9a de .ai/specs/app-android/spec.md).
 *
 * Guarda pares nome e valor cifrados com AES-GCM por uma chave que nasce e fica dentro do Android Keystore
 * (o app nunca ve os bytes da chave). O que vai ao disco e so o texto cifrado, em SharedPreferences ("cofre.xml"),
 * que o manifest deixa fora do backup. Tres operacoes e nada mais: ler, gravar e apagar.
 *
 * Quem usa e web/js/sessao.js: o refresh token do Cognito e, durante um login, o verifier e o state do PKCE.
 * Este plugin nao escreve log: nem nome, nem valor, nem erro com conteudo.
 *
 * Limite conhecido (R18i): "ler" responde ao JavaScript da WebView, entao um script injetado na pagina obtem o valor.
 * O ganho e o arquivo cifrado em disco e fora do backup.
 */
@CapacitorPlugin(name = "Cofre")
public class CofrePlugin extends Plugin {

    private static final String ARQUIVO = "cofre";
    private static final String CHAVE = "financas.cofre";
    private static final String KEYSTORE = "AndroidKeyStore";
    private static final String CIFRA = "AES/GCM/NoPadding";
    private static final int IV_BYTES = 12;
    private static final int TAG_BITS = 128;

    private SharedPreferences arquivo() {
        return getContext().getSharedPreferences(ARQUIVO, Context.MODE_PRIVATE);
    }

    private SecretKey chave() throws Exception {
        KeyStore ks = KeyStore.getInstance(KEYSTORE);
        ks.load(null);
        if (ks.containsAlias(CHAVE)) {
            return ((KeyStore.SecretKeyEntry) ks.getEntry(CHAVE, null)).getSecretKey();
        }
        KeyGenerator gerador = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE);
        gerador.init(new KeyGenParameterSpec.Builder(CHAVE, KeyProperties.PURPOSE_ENCRYPT | KeyProperties.PURPOSE_DECRYPT)
            .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
            .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
            .setKeySize(256)
            .build());
        return gerador.generateKey();
    }

    /** Devolve iv + texto cifrado em Base64. O iv e sorteado pelo Keystore a cada gravacao. O nome entra como dado autenticado. */
    private String cifrar(String nome, String valor) throws Exception {
        Cipher c = Cipher.getInstance(CIFRA);
        c.init(Cipher.ENCRYPT_MODE, chave());
        c.updateAAD(nome.getBytes(StandardCharsets.UTF_8));
        byte[] iv = c.getIV();
        byte[] cifrado = c.doFinal(valor.getBytes(StandardCharsets.UTF_8));
        byte[] tudo = new byte[iv.length + cifrado.length];
        System.arraycopy(iv, 0, tudo, 0, iv.length);
        System.arraycopy(cifrado, 0, tudo, iv.length, cifrado.length);
        return Base64.encodeToString(tudo, Base64.NO_WRAP);
    }

    private String decifrar(String nome, String guardado) throws Exception {
        byte[] tudo = Base64.decode(guardado, Base64.NO_WRAP);
        Cipher c = Cipher.getInstance(CIFRA);
        c.init(Cipher.DECRYPT_MODE, chave(), new GCMParameterSpec(TAG_BITS, tudo, 0, IV_BYTES));
        c.updateAAD(nome.getBytes(StandardCharsets.UTF_8));
        return new String(c.doFinal(tudo, IV_BYTES, tudo.length - IV_BYTES), StandardCharsets.UTF_8);
    }

    private String nome(PluginCall call) {
        String nome = call.getString("nome");
        if (nome == null || nome.isEmpty()) {
            call.reject("Cofre: falta o nome.");
            return null;
        }
        return nome;
    }

    /** ler({nome}) devolve {valor}; valor e null quando nao ha nada guardado com esse nome. */
    @PluginMethod
    public void ler(PluginCall call) {
        String nome = nome(call);
        if (nome == null) return;
        try {
            String guardado = arquivo().getString(nome, null);
            JSObject r = new JSObject();
            r.put("valor", guardado == null ? JSObject.NULL : decifrar(nome, guardado));
            call.resolve(r);
        } catch (Exception e) {
            call.reject("Cofre: não consegui ler.");
        }
    }

    /** gravar({nome, valor}) so responde depois de o arquivo estar gravado. */
    @PluginMethod
    public void gravar(PluginCall call) {
        String nome = nome(call);
        if (nome == null) return;
        String valor = call.getString("valor");
        if (valor == null || valor.isEmpty()) {
            call.reject("Cofre: falta o valor.");
            return;
        }
        try {
            if (!arquivo().edit().putString(nome, cifrar(nome, valor)).commit()) {
                call.reject("Cofre: não consegui gravar.");
                return;
            }
            call.resolve();
        } catch (Exception e) {
            call.reject("Cofre: não consegui gravar.");
        }
    }

    /** apagar({nome}) so responde depois de o arquivo estar gravado sem o nome. Apagar o que nao existe da certo. */
    @PluginMethod
    public void apagar(PluginCall call) {
        String nome = nome(call);
        if (nome == null) return;
        if (!arquivo().edit().remove(nome).commit()) {
            call.reject("Cofre: não consegui apagar.");
            return;
        }
        call.resolve();
    }
}
