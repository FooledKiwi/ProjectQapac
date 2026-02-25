package com.fooledkiwi.projectqapacapp.activities;

import android.Manifest;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.view.View;

import androidx.activity.EdgeToEdge;
import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.content.ContextCompat;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.session.SessionManager;

public class FirstTimeActivity extends AppCompatActivity {

    private ActivityResultLauncher<String[]> locationPermissionLauncher;

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

        locationPermissionLauncher = registerForActivityResult(
                new ActivityResultContracts.RequestMultiplePermissions(), result -> {});

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
        requestLocationPermissionIfNeeded();
        startActivity(intent);

    }

    public void gotoMain() {
        Intent intent = new Intent(this, MainActivity.class);
        requestLocationPermissionIfNeeded();
        startActivity(intent);
    }

    private void requestLocationPermissionIfNeeded() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION)
                != PackageManager.PERMISSION_GRANTED) {
            locationPermissionLauncher.launch(new String[]{
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
            });
        }
    }
}