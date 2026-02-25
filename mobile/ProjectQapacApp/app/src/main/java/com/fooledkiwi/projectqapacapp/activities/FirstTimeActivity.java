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

    // Destino pendiente mientras se espera la respuesta del permiso.
    // "main" → MainActivity, "auth" → AuthActivity (pendingAction lleva "login"/"register")
    private String pendingDestination = null;
    private String pendingAction      = null;

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
                new ActivityResultContracts.RequestMultiplePermissions(), result -> {
                    Boolean fine   = result.get(Manifest.permission.ACCESS_FINE_LOCATION);
                    Boolean coarse = result.get(Manifest.permission.ACCESS_COARSE_LOCATION);
                    boolean granted = (fine != null && fine) || (coarse != null && coarse);

                    if (granted && pendingDestination != null) {
                        executePendingNavigation();
                    }
                    // Si denegado: no navegar, el usuario queda en FirstTimeActivity.
                    pendingDestination = null;
                    pendingAction      = null;
                });

        // Si el usuario ya tiene sesión activa, ir directo a MainActivity
        SessionManager session = new SessionManager(this);
        if (session.isLoggedIn()) {
            gotoMain();
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
        if (hasLocationPermission()) {
            startActivity(buildAuthIntent(action));
        } else {
            pendingDestination = "auth";
            pendingAction      = action;
            locationPermissionLauncher.launch(new String[]{
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
            });
        }
    }

    public void gotoMain() {
        if (hasLocationPermission()) {
            startActivity(new Intent(this, MainActivity.class));
        } else {
            pendingDestination = "main";
            pendingAction      = null;
            locationPermissionLauncher.launch(new String[]{
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
            });
        }
    }

    // -------------------------------------------------------------------------

    private boolean hasLocationPermission() {
        return ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION)
                == PackageManager.PERMISSION_GRANTED
                || ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_COARSE_LOCATION)
                == PackageManager.PERMISSION_GRANTED;
    }

    private Intent buildAuthIntent(String action) {
        Intent intent = new Intent(this, AuthActivity.class);
        intent.putExtra("TAB_POS", action);
        return intent;
    }

    private void executePendingNavigation() {
        if ("auth".equals(pendingDestination)) {
            startActivity(buildAuthIntent(pendingAction));
        } else if ("main".equals(pendingDestination)) {
            startActivity(new Intent(this, MainActivity.class));
        }
    }
}
