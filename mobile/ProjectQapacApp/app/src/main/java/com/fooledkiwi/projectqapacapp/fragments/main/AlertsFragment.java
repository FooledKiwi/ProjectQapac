package com.fooledkiwi.projectqapacapp.fragments.main;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;

import androidx.fragment.app.Fragment;
import androidx.recyclerview.widget.LinearLayoutManager;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.adapters.AlertAdapter;
import com.fooledkiwi.projectqapacapp.models.Alert;

import java.util.Arrays;
import java.util.List;

public class AlertsFragment extends Fragment {

    public AlertsFragment() {
        // Required empty public constructor
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container,
                             Bundle savedInstanceState) {
        return inflater.inflate(R.layout.fragment_alerts, container, false);
    }

    @Override
    public void onViewCreated(View view, Bundle savedInstanceState) {
        super.onViewCreated(view, savedInstanceState);

        RecyclerView rvAllAlertas = view.findViewById(R.id.rvAllAlertas);
        TextView tvCount = view.findViewById(R.id.tvCountCurrentAlerts);

        List<Alert> alerts = Arrays.asList(
                new Alert("Cambio de Ruta",     "RUTA P13 - Placa: R2DX-145", "Hace 3m"),
                new Alert("Desvío Temporal",    "RUTA P7  - Placa: A1BC-234", "Hace 15m"),
                new Alert("Bus Fuera de Línea", "RUTA P2  - Placa: Z9XY-001", "Hace 32m"),
                new Alert("Retraso Esperado",   "RUTA P5  - Placa: K3LM-789", "Hace 1h"),
                new Alert("Servicio Reanudado", "RUTA P1  - Placa: B7QR-456", "Hace 2h")
        );

        AlertAdapter adapter = new AlertAdapter(alerts);
        rvAllAlertas.setLayoutManager(new LinearLayoutManager(getContext()));
        rvAllAlertas.setAdapter(adapter);

        tvCount.setText(String.valueOf(alerts.size()));
    }
}
