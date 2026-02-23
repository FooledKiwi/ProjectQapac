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
        holder.btnFavorite.setOnCheckedChangeListener((buttonView, isChecked) -> onFavoriteClick(stop));
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
        CheckBox btnFavorite;

        public StopViewHolder(@NonNull View itemView) {
            super(itemView);
            tvRouteName = itemView.findViewById(R.id.tvRouteName);
            tvLabelVehicle = itemView.findViewById(R.id.tvLabelVehicle);
            btnFavorite = itemView.findViewById(R.id.btnFavorite);
        }
    }
}
