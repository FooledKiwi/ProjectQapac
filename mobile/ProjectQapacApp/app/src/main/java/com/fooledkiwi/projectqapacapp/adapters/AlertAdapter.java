package com.fooledkiwi.projectqapacapp.adapters;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.models.Alert;

import java.util.List;

public class AlertAdapter extends RecyclerView.Adapter<AlertAdapter.AlertViewHolder> {

    private final List<Alert> alerts;

    public AlertAdapter(List<Alert> alerts) {
        this.alerts = alerts;
    }

    @NonNull
    @Override
    public AlertViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View view = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.ly_minor_alerts, parent, false);
        return new AlertViewHolder(view);
    }

    @Override
    public void onBindViewHolder(@NonNull AlertViewHolder holder, int position) {
        Alert alert = alerts.get(position);
        holder.tvTitleNotification.setText(alert.getTitle());
        holder.tvSubtitleNotification.setText(alert.getSubtitle());
        holder.tvTimeAlert.setText(alert.getTime());
    }

    @Override
    public int getItemCount() {
        return alerts.size();
    }

    public static class AlertViewHolder extends RecyclerView.ViewHolder {

        TextView tvTitleNotification;
        TextView tvSubtitleNotification;
        TextView tvTimeAlert;

        public AlertViewHolder(@NonNull View itemView) {
            super(itemView);
            tvTitleNotification   = itemView.findViewById(R.id.tvTitleNotification);
            tvSubtitleNotification = itemView.findViewById(R.id.tvSubtitleNotification);
            tvTimeAlert           = itemView.findViewById(R.id.tvTimeAlert);
        }
    }
}
