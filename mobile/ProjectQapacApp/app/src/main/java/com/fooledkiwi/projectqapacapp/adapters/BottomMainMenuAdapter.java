package com.fooledkiwi.projectqapacapp.adapters;

import androidx.fragment.app.Fragment;
import androidx.fragment.app.FragmentActivity;
import androidx.viewpager2.adapter.FragmentStateAdapter;

import com.fooledkiwi.projectqapacapp.fragments.main.AlertsFragment;
import com.fooledkiwi.projectqapacapp.fragments.main.ExploreFragment;
import com.fooledkiwi.projectqapacapp.fragments.main.HistoryFragment;
import com.fooledkiwi.projectqapacapp.fragments.main.RatingFragment;

import org.jetbrains.annotations.NotNull;

public class BottomMainMenuAdapter extends FragmentStateAdapter {
    public BottomMainMenuAdapter(@NotNull FragmentActivity fragmentActivity) {
        super(fragmentActivity);
    }

    @NotNull
    @Override
    public Fragment createFragment(int pos) {
        if(pos == 0) {return  new ExploreFragment();}
        else if(pos == 1) return new HistoryFragment();
        else if(pos == 2) return new AlertsFragment();
        else if(pos == 3) return new RatingFragment();
        else  return new ExploreFragment();
    }

    @Override
    public int getItemCount() {
        return 4;
    }
}
