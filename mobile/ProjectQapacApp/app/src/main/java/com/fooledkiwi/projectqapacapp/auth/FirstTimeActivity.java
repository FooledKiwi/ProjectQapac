package com.fooledkiwi.projectqapacapp.auth;

import android.content.Intent;
import android.content.SharedPreferences;
import android.os.Bundle;
import android.view.View;

import androidx.activity.EdgeToEdge;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;

import com.fooledkiwi.projectqapacapp.MainActivity;
import com.fooledkiwi.projectqapacapp.R;

public class FirstTimeActivity extends AppCompatActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        EdgeToEdge.enable(this);
        setContentView(R.layout.activity_fist_time);
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.main), (v, insets) -> {
            Insets systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars());
            v.setPadding(systemBars.left, systemBars.top, systemBars.right, systemBars.bottom);
            return insets;
        });

        if(!checkIfFirstTime()) {
            gotoMain();
            /// Acá va a iniciar sesión automáticamente.
            finish();
        }
    }

    public void registerBtn(View vw) {
        gotoAuth("register");
    }

    public void loginBtn(View vw) {
        gotoAuth("login");
    }

    public void enterAsGuest(View vw) {

    }

    public void gotoAuth(String action) {
        Intent intent = new Intent(this, AuthActivity.class);
        intent.putExtra("TAB_POS", action);
        startActivity(intent);
    }

    public void gotoMain() {
        Intent intent = new Intent(this, MainActivity.class);
        startActivity(intent);
    }

    public boolean checkIfFirstTime() {
        SharedPreferences prefs = getSharedPreferences("QapacPrefs", MODE_PRIVATE);
        return prefs.getBoolean("first_time", true);
    }
}