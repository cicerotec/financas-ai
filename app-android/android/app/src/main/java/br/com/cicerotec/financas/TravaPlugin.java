package br.com.cicerotec.financas;

import android.view.WindowManager;

import androidx.annotation.NonNull;
import androidx.biometric.BiometricManager;
import androidx.biometric.BiometricPrompt;
import androidx.core.content.ContextCompat;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * Trava: a confirmacao de quem esta com o aparelho na mao (item 54, fase 3; regras R19 a R23 de .ai/specs/app-android/spec.md).
 *
 * Tres operacoes: disponivel (o aparelho tem digital, rosto ou senha de tela?), confirmar (abre a janela do Android)
 * e protegerJanela (marca a janela como protegida: o app some dos apps recentes e nao aceita captura nem gravacao de tela).
 *
 * Quem decide quando bloquear e web/js/bloqueio.js. Este plugin so pergunta ao Android e devolve a resposta: nao guarda
 * a escolha da pessoa, nao cifra nada e nao escreve log. A trava e so da tela (R23).
 */
@CapacitorPlugin(name = "Trava")
public class TravaPlugin extends Plugin {

    // Digital ou rosto (qualquer classe que o aparelho aceite) ou a credencial do aparelho (PIN, padrao ou senha).
    private static final int QUEM = BiometricManager.Authenticators.BIOMETRIC_WEAK | BiometricManager.Authenticators.DEVICE_CREDENTIAL;

    private boolean pode() {
        return BiometricManager.from(getContext()).canAuthenticate(QUEM) == BiometricManager.BIOMETRIC_SUCCESS;
    }

    /** disponivel() devolve {disponivel}: falso quando o aparelho nao tem digital, rosto nem senha de tela. */
    @PluginMethod
    public void disponivel(PluginCall call) {
        JSObject r = new JSObject();
        r.put("disponivel", pode());
        call.resolve(r);
    }

    /**
     * confirmar({titulo}) so resolve com a confirmacao do Android. Recusa com o codigo "indisponivel" (sem bloqueio de tela),
     * "bloqueado" (muitas tentativas) ou "cancelado" (a pessoa fechou a janela, ou qualquer outro erro).
     */
    @PluginMethod
    public void confirmar(final PluginCall call) {
        if (!pode()) {
            call.reject("Trava: o aparelho não tem bloqueio de tela.", "indisponivel");
            return;
        }
        final String titulo = call.getString("titulo", "Confirme que é você");
        getActivity().runOnUiThread(() -> {
            try {
                BiometricPrompt.PromptInfo info = new BiometricPrompt.PromptInfo.Builder()
                    .setTitle(titulo)
                    .setAllowedAuthenticators(QUEM)
                    .build();
                new BiometricPrompt(getActivity(), ContextCompat.getMainExecutor(getContext()), new BiometricPrompt.AuthenticationCallback() {
                    @Override
                    public void onAuthenticationSucceeded(@NonNull BiometricPrompt.AuthenticationResult resultado) {
                        call.resolve();
                    }

                    @Override
                    public void onAuthenticationError(int codigo, @NonNull CharSequence texto) {
                        boolean muitas = codigo == BiometricPrompt.ERROR_LOCKOUT || codigo == BiometricPrompt.ERROR_LOCKOUT_PERMANENT;
                        call.reject("Trava: não confirmado.", muitas ? "bloqueado" : "cancelado");
                    }
                    // onAuthenticationFailed (digital nao reconhecida) nao encerra: a janela do Android continua aberta.
                }).authenticate(info);
            } catch (Exception e) {
                call.reject("Trava: não consegui abrir a confirmação.", "cancelado");
            }
        });
    }

    /** protegerJanela({ligar}) liga ou desliga FLAG_SECURE na janela do app. */
    @PluginMethod
    public void protegerJanela(final PluginCall call) {
        final boolean ligar = Boolean.TRUE.equals(call.getBoolean("ligar", false));
        getActivity().runOnUiThread(() -> {
            if (ligar) getActivity().getWindow().addFlags(WindowManager.LayoutParams.FLAG_SECURE);
            else getActivity().getWindow().clearFlags(WindowManager.LayoutParams.FLAG_SECURE);
            call.resolve();
        });
    }
}
