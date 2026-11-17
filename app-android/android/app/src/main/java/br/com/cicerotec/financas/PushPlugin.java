package br.com.cicerotec.financas;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.content.Context;
import android.os.Build;

import androidx.core.app.NotificationManagerCompat;

import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;
import com.google.firebase.FirebaseApp;
import com.google.firebase.messaging.FirebaseMessaging;

/**
 * Push: o aparelho recebe os avisos do app por notificacao (item 54, fase 4; regras R14 a R17 e R19 de
 * .ai/specs/app-android-push/spec.md).
 *
 * Operacoes: disponivel, permissao, pedirPermissao, token, apagarToken, mudou e marcarRegistrado. O SDK do Firebase Messaging
 * (do Google) obtem o token; este plugin so o repassa ao JavaScript. Cria o canal de notificacao "avisos" ao carregar.
 *
 * Quem decide o que fazer com o token (registrar no servidor, quando registrar de novo, o que dizer) e o JavaScript
 * (web/js). O servidor decide quem recebe e o que a notificacao diz. Este plugin nao escreve log: nem o token, nem erro com
 * conteudo (R23). O token so aparece para o JavaScript do app e nunca e gravado aqui.
 */
@CapacitorPlugin(
    name = "Push",
    permissions = { @Permission(alias = "notificacoes", strings = { "android.permission.POST_NOTIFICATIONS" }) }
)
public class PushPlugin extends Plugin {

    /** O canal que a mensagem do servidor pede (channel_id em backend/adapter/notificacao/fcm.go). */
    static final String CANAL_AVISOS = "avisos";
    /** Marca "o token mudou", gravada por FinancasMessagingService.onNewToken e lida por mudou(). So um booleano, nunca o token. */
    static final String ARQUIVO = "push";
    static final String MUDOU = "mudou";

    @Override
    public void load() {
        criarCanal(getContext());
    }

    /** Cria o canal "avisos" (importancia padrao). Criar de novo um canal que existe nao muda nada. */
    static void criarCanal(Context contexto) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;
        NotificationChannel canal = new NotificationChannel(CANAL_AVISOS, "Avisos", NotificationManager.IMPORTANCE_DEFAULT);
        canal.setDescription("Contas atrasadas, que vencem hoje e a vencer.");
        NotificationManager gerente = contexto.getSystemService(NotificationManager.class);
        if (gerente != null) gerente.createNotificationChannel(canal);
    }

    /**
     * O Firebase so esta pronto se o build levou o google-services.json (sem ele, FirebaseApp nao inicializa; P2) e se o aparelho tem
     * os Google Play Services. Aqui so se confere que o pacote existe: se estiver desatualizado ou desligado, token() recusa.
     */
    private boolean firebasePronto() {
        try {
            getContext().getPackageManager().getPackageInfo("com.google.android.gms", 0);
            return !FirebaseApp.getApps(getContext()).isEmpty();
        } catch (Exception e) {
            return false;
        }
    }

    /** "concedida" quando o Android deixa o app notificar; "pergunte" se ainda nunca perguntou (Android 13 ou mais); senao "negada". */
    private String estadoDaPermissao() {
        if (NotificationManagerCompat.from(getContext()).areNotificationsEnabled()) return "concedida";
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU && getPermissionState("notificacoes") == PermissionState.PROMPT) {
            return "pergunte";
        }
        return "negada";
    }

    private void responderPermissao(PluginCall call) {
        JSObject r = new JSObject();
        r.put("permissao", estadoDaPermissao());
        call.resolve(r);
    }

    /** disponivel() devolve {disponivel}: falso sem Google Play Services ou sem google-services.json no build. */
    @PluginMethod
    public void disponivel(PluginCall call) {
        JSObject r = new JSObject();
        r.put("disponivel", firebasePronto());
        call.resolve(r);
    }

    /** permissao() devolve {permissao}: "concedida", "pergunte" ou "negada". Nao abre janela nenhuma. */
    @PluginMethod
    public void permissao(PluginCall call) {
        responderPermissao(call);
    }

    /**
     * pedirPermissao() abre o pedido do Android (so Android 13 ou mais, e so se ainda nunca foi negado de vez) e devolve {permissao}.
     * Antes do Android 13 nada e pedido: a permissao ja vale, a menos que a pessoa tenha desligado as notificacoes do app.
     */
    @PluginMethod
    public void pedirPermissao(PluginCall call) {
        if ("pergunte".equals(estadoDaPermissao())) {
            requestPermissionForAlias("notificacoes", call, "aoPedir");
            return;
        }
        responderPermissao(call);
    }

    @PermissionCallback
    private void aoPedir(PluginCall call) {
        responderPermissao(call);
    }

    /** token() devolve {token}. Recusa com o codigo "indisponivel" quando o Firebase nao esta pronto ou nao entrega. */
    @PluginMethod
    public void token(final PluginCall call) {
        if (!firebasePronto()) {
            call.reject("Push: indisponível neste aparelho.", "indisponivel");
            return;
        }
        try {
            FirebaseMessaging.getInstance().getToken().addOnCompleteListener(tarefa -> {
                String token = tarefa.isSuccessful() ? tarefa.getResult() : null;
                if (token == null || token.isEmpty()) {
                    call.reject("Push: não consegui o token.", "indisponivel");
                    return;
                }
                JSObject r = new JSObject();
                r.put("token", token);
                call.resolve(r);
            });
        } catch (Exception e) {
            call.reject("Push: não consegui o token.", "indisponivel");
        }
    }

    /** apagarToken() invalida o token no FCM (o Sair, R19): o servidor o apaga na proxima rotina. Sem Firebase, nada a apagar. */
    @PluginMethod
    public void apagarToken(final PluginCall call) {
        if (!firebasePronto()) {
            call.resolve();
            return;
        }
        try {
            FirebaseMessaging.getInstance().deleteToken().addOnCompleteListener(tarefa -> {
                if (tarefa.isSuccessful()) call.resolve();
                else call.reject("Push: não consegui apagar o token.");
            });
        } catch (Exception e) {
            call.reject("Push: não consegui apagar o token.");
        }
    }

    /** mudou() devolve {mudou}: verdadeiro se o Firebase trocou o token desde o ultimo marcarRegistrado() (R15). */
    @PluginMethod
    public void mudou(PluginCall call) {
        JSObject r = new JSObject();
        r.put("mudou", getContext().getSharedPreferences(ARQUIVO, Context.MODE_PRIVATE).getBoolean(MUDOU, false));
        call.resolve(r);
    }

    /** marcarRegistrado() limpa a marca "o token mudou", depois de o JavaScript registrar o token novo no servidor. */
    @PluginMethod
    public void marcarRegistrado(PluginCall call) {
        if (!getContext().getSharedPreferences(ARQUIVO, Context.MODE_PRIVATE).edit().putBoolean(MUDOU, false).commit()) {
            call.reject("Push: não consegui gravar.");
            return;
        }
        call.resolve();
    }
}
