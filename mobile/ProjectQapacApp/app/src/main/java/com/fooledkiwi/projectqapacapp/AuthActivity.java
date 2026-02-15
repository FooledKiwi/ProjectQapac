package com.fooledkiwi.projectqapacapp;

import android.os.Bundle;

import androidx.activity.EdgeToEdge;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowInsetsCompat;
import androidx.viewpager2.widget.ViewPager2;

import com.fooledkiwi.projectqapacapp.fragments.auth.TabAuthAdapter;
import com.google.android.material.tabs.TabLayout;
import com.google.android.material.tabs.TabLayoutMediator;

public class AuthActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        EdgeToEdge.enable(this);
        setContentView(R.layout.activity_auth);
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.main), (v, insets) -> {
            Insets systemBars = insets.getInsets(WindowInsetsCompat.Type.systemBars());
            v.setPadding(systemBars.left, systemBars.top, systemBars.right, systemBars.bottom);
            return insets;
        });

        loadTabAuthFragments();
    }

    public void loadTabAuthFragments() {
        TabLayout authTab = findViewById(R.id.tablay_auth);
        ViewPager2 viewPager = findViewById(R.id.vp2_authtab);
        TabAuthAdapter adapter = new TabAuthAdapter(this);
        viewPager.setAdapter(adapter);

        new TabLayoutMediator(authTab, viewPager, (tab, pos) -> {
            if (pos == 0) {
                tab.setText(getString(R.string.login));
            }else tab.setText(getString(R.string.register));
        }).attach();
    }
}