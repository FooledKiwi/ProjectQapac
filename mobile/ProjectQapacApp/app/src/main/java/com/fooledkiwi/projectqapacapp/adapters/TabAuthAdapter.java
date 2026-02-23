package com.fooledkiwi.projectqapacapp.adapters;

import androidx.annotation.NonNull;
import androidx.fragment.app.Fragment;
import androidx.fragment.app.FragmentActivity;
import androidx.viewpager2.adapter.FragmentStateAdapter;

import com.fooledkiwi.projectqapacapp.fragments.auth.AuthLoginFragment;
import com.fooledkiwi.projectqapacapp.fragments.auth.AuthRegisterFragment;

public class TabAuthAdapter extends FragmentStateAdapter {
    public TabAuthAdapter(@NonNull FragmentActivity fragmentActivity) {
        super(fragmentActivity);
    }

    @NonNull
    @Override
    public Fragment createFragment(int pos) {
        if(pos == 0) {
            return new AuthLoginFragment();
        } else {
            return new AuthRegisterFragment();
        }
    }

    @Override
    public int getItemCount() {
        return 2;
    }
}
