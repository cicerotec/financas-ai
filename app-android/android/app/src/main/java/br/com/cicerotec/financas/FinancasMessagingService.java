package br.com.cicerotec.financas;

import android.content.Context;

import androidx.annotation.NonNull;

import com.google.firebase.messaging.FirebaseMessagingService;
import com.google.firebase.messaging.RemoteMessage;

/**
 * O servico que o SDK do Firebase Messaging exige (item 54, fase 4; R14 de .ai/specs/app-android-push/spec.md).
 *
 * Duas coisas e nada mais:
 *  - onNewToken so marca que o token mudou (um booleano); o JavaScript registra o token novo no servidor quando o app abre (R15).
 *    O token nao e gravado aqui e nao vai a log.
 *  - onMessageReceived ignora a mensagem: com o app em segundo plano ou fechado o proprio Android mostra a notificacao do servidor
 *    (mensagem de notificacao), e com o app aberto a pessoa ja esta olhando para ele (P3).
 */
public class FinancasMessagingService extends FirebaseMessagingService {

    @Override
    public void onCreate() {
        super.onCreate();
        PushPlugin.criarCanal(this); // o servico pode acordar antes de o app abrir: o canal precisa existir
    }

    @Override
    public void onNewToken(@NonNull String token) {
        getSharedPreferences(PushPlugin.ARQUIVO, Context.MODE_PRIVATE).edit().putBoolean(PushPlugin.MUDOU, true).apply();
    }

    @Override
    public void onMessageReceived(@NonNull RemoteMessage mensagem) {
        // de proposito vazio (ver acima)
    }
}
