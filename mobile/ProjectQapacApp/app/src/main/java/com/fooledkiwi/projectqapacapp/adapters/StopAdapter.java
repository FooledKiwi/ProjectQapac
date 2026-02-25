package com.fooledkiwi.projectqapacapp.adapters;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.CheckBox;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.models.Stop;

import java.util.List;
import java.util.Locale;

public class StopAdapter extends RecyclerView.Adapter<StopAdapter.StopViewHolder> {

    private final List<Stop> stops;

    public StopAdapter(List<Stop> stops) {
        this.stops = stops;
    }

    @NonNull
    @Override
    public StopViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View view = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.ly_stop, parent, false);
        return new StopViewHolder(view);
    }

    @Override
    public void onBindViewHolder(@NonNull StopViewHolder holder, int position) {
        Stop stop = stops.get(position);
        holder.tvRouteName.setText(stop.getName());
        holder.tvLabelVehicle.setText(stop.getLat() + ", " + stop.getLon());
        holder.tvEtaSeconds.setText(formatEta(stop.getEtaSeconds()));
        holder.btnFavorite.setOnCheckedChangeListener((buttonView, isChecked) -> onFavoriteClick(stop));
    }

    private String formatEta(int etaSeconds) {
        if (etaSeconds <= 0) return "--:--";
        int minutes = etaSeconds / 60;
        int seconds = etaSeconds % 60;
        return String.format(Locale.getDefault(), "%02d:%02d", minutes, seconds);
    }

    void onFavoriteClick(Stop stop) {
        // TODO: implementar lógica de favorito
    }

    @Override
    public int getItemCount() {
        return stops.size();
    }

    public static class StopViewHolder extends RecyclerView.ViewHolder {

        TextView tvRouteName;
        TextView tvLabelVehicle;
        TextView tvEtaSeconds;
        CheckBox btnFavorite;

        public StopViewHolder(@NonNull View itemView) {
            super(itemView);
            tvRouteName    = itemView.findViewById(R.id.tvRouteName);
            tvLabelVehicle = itemView.findViewById(R.id.tvLabelVehicle);
            tvEtaSeconds   = itemView.findViewById(R.id.tvEtaSeconds);
            btnFavorite    = itemView.findViewById(R.id.btnFavorite);
        }
    }
}
