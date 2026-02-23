package com.fooledkiwi.projectqapacapp.activities;

import android.Manifest;
import android.content.pm.PackageManager;
import android.os.Bundle;
import android.widget.Toast;

import androidx.activity.EdgeToEdge;
import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.content.ContextCompat;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;
import androidx.viewpager2.widget.ViewPager2;

import com.fooledkiwi.projectqapacapp.adapters.BottomMainMenuAdapter;
import com.fooledkiwi.projectqapacapp.R;
import com.google.android.material.bottomnavigation.BottomNavigationView;

public class MainActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        EdgeToEdge.enable(this);
        setContentView(R.layout.activity_main);
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.main), (v, insets) -> {
            Insets systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars());
            v.setPadding(systemBars.left, systemBars.top, systemBars.right, systemBars.bottom);
            return insets;
        });

        loadBottomMenuInteraction();

        if (ContextCompat.checkSelfPermission(this, Manifest.permission.ACCESS_FINE_LOCATION) != PackageManager.PERMISSION_GRANTED) {
            requestPermissionLauncher.launch(new String[]{
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
            });
        }
    }

    private final ActivityResultLauncher<String[]> requestPermissionLauncher =
        registerForActivityResult(new ActivityResultContracts.RequestMultiplePermissions(), result -> {
            Boolean fineGranted = result.getOrDefault(Manifest.permission.ACCESS_FINE_LOCATION, false);
            Boolean coarseGranted = result.getOrDefault(Manifest.permission.ACCESS_COARSE_LOCATION, false);

            if ((fineGranted != null && fineGranted) || (coarseGranted != null && coarseGranted)) {
                // El usuario aceptó el permiso en la ventana emergente
                Toast.makeText(this, "Gracias por compartir tu ubicación", Toast.LENGTH_LONG).show();
            } else {
                // El usuario denegó el permiso
                Toast.makeText(this, "Necesitamos tu ubicación para mostrarte el mapa correctamente", Toast.LENGTH_LONG).show();
            }
        });

    public void loadBottomMenuInteraction() {
        ViewPager2 viewPager = findViewById(R.id.vp2_mainPager);
        BottomNavigationView bottomNav = findViewById(R.id.bottomNav);
        BottomMainMenuAdapter adapter = new BottomMainMenuAdapter(this);
        viewPager.setAdapter(adapter);
        viewPager.setUserInputEnabled(false);

        bottomNav.setOnItemSelectedListener(item -> {
            int itemId = item.getItemId();
            if (itemId == R.id.nav_explorar) {
                viewPager.setCurrentItem(0, true);
                return true;
            } else if (itemId == R.id.nav_historial) {
                viewPager.setCurrentItem(1, true);
                return true;
            } else if (itemId == R.id.nav_alerts) {
                viewPager.setCurrentItem(2, true);
                return true;
            } else if (itemId == R.id.nav_calificar) {
                viewPager.setCurrentItem(3, true);
                return true;
            }
            else
            return false;
        });

        viewPager.registerOnPageChangeCallback(new ViewPager2.OnPageChangeCallback() {
            @Override
            public void onPageSelected(int position) {
                super.onPageSelected(position);
                switch (position) {
                    case 0:
                        bottomNav.setSelectedItemId(R.id.nav_explorar);
                        break;
                    case 1:
                        bottomNav.setSelectedItemId(R.id.nav_historial);
                        break;
                    case 2:
                        bottomNav.setSelectedItemId(R.id.nav_alerts);
                        break;
                    case 3:
                        bottomNav.setSelectedItemId(R.id.nav_calificar);
                        break;
                }
            }
        });
    }
}