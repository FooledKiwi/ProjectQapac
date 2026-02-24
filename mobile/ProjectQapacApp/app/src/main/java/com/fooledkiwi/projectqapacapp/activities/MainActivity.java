package com.fooledkiwi.projectqapacapp.activities;

import android.os.Bundle;

import androidx.activity.EdgeToEdge;
import androidx.appcompat.app.AppCompatActivity;
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
    }

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
            } else if (itemId == R.id.nav_cuenta) {
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
                        bottomNav.setSelectedItemId(R.id.nav_cuenta);
                        break;
                }
            }
        });
    }
}