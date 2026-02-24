package com.fooledkiwi.projectqapacapp.fragments.main;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;
import androidx.recyclerview.widget.LinearLayoutManager;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.adapters.TravelAdapter;
import com.fooledkiwi.projectqapacapp.models.Travel;

import java.util.ArrayList;
import java.util.List;

public class HistoryFragment extends Fragment {

    public HistoryFragment() {
        // Required empty public constructor
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        return inflater.inflate(R.layout.fragment_history, container, false);
    }

    @Override
    public void onViewCreated(@NonNull View view, @Nullable Bundle savedInstanceState) {
        super.onViewCreated(view, savedInstanceState);

        List<Travel> travels = new ArrayList<>();
        travels.add(new Travel("RUTA P13", "DCM-1519", "25 min", 4.5f, "Juan Rodríguez"));
        travels.add(new Travel("RUTA P07", "BCX-3342", "15 min", 3.0f, "Carlos Pérez"));
        travels.add(new Travel("RUTA P21", "AKL-8871", "40 min", 5.0f, "Miguel Torres"));

        RecyclerView rvHistory = view.findViewById(R.id.rvHistory);
        rvHistory.setLayoutManager(new LinearLayoutManager(requireContext()));
        rvHistory.setAdapter(new TravelAdapter(travels));
    }
}
