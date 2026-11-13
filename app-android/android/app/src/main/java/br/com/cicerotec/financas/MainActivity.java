package br.com.cicerotec.financas;

import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.webkit.WebView;

import androidx.core.content.ContextCompat;
import androidx.core.splashscreen.SplashScreen;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.WebViewListener;

public class MainActivity extends BridgeActivity {

    // O splash do Android (o icone no centro) fica ate a pagina carregar; sem isso ele sai antes e sobra uma tela vazia.
    private static final long ESPERA_MAXIMA_MS = 4000;
    private volatile boolean paginaPronta = false;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        SplashScreen splash = SplashScreen.installSplashScreen(this);
        splash.setKeepOnScreenCondition(() -> !paginaPronta);
        // O Cofre (armazenamento do login, cifrado pelo Android Keystore) e um plugin nosso: precisa ser registrado antes do super.
        registerPlugin(CofrePlugin.class);
        // A Trava (biometria e janela protegida) tambem e nossa.
        registerPlugin(TravaPlugin.class);
        super.onCreate(savedInstanceState);
        // A pagina leva um instante para desenhar: ate la a WebView mostra o fundo do app, e nao um quadro preto.
        getBridge().getWebView().setBackgroundColor(ContextCompat.getColor(this, R.color.fundo));
        getBridge().addWebViewListener(new WebViewListener() {
            @Override
            public void onPageLoaded(WebView webView) {
                paginaPronta = true; // a Abertura (web/js/abertura.js) ja esta desenhada, com o mesmo icone no mesmo lugar
            }
        });
        // Se a pagina nao avisar (falha ao carregar), o splash sai sozinho e o que houver aparece.
        new Handler(Looper.getMainLooper()).postDelayed(() -> paginaPronta = true, ESPERA_MAXIMA_MS);
    }
}
