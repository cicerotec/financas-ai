package br.com.cicerotec.financas;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {

    @Override
    public void onCreate(Bundle savedInstanceState) {
        // O Cofre (armazenamento do login, cifrado pelo Android Keystore) e um plugin nosso: precisa ser registrado antes do super.
        registerPlugin(CofrePlugin.class);
        super.onCreate(savedInstanceState);
    }
}
