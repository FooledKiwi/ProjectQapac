package com.fooledkiwi.projectqapacapp.activities;

import android.content.Intent;
import android.os.Bundle;
import android.view.View;

import androidx.activity.EdgeToEdge;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.session.SessionManager;

public class FirstTimeActivity extends AppCompatActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        EdgeToEdge.enable(this);
        setContentView(R.layout.activity_first_time);
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.main), (v, insets) -> {
            Insets systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars());
            v.setPadding(systemBars.left, systemBars.top, systemBars.right, systemBars.bottom);
            return insets;
        });

        // Si el usuario ya tiene sesión activa, ir directo a MainActivity
        SessionManager session = new SessionManager(this);
        if (session.isLoggedIn()) {
            gotoMain();
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
        gotoMain();
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
}