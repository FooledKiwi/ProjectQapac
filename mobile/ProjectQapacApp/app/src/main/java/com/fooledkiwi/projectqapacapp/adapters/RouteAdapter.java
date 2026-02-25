package com.fooledkiwi.projectqapacapp.adapters;

import android.content.res.ColorStateList;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.cardview.widget.CardView;
import androidx.core.content.ContextCompat;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.models.Route;

import java.util.List;

public class RouteAdapter extends RecyclerView.Adapter<RouteAdapter.RouteViewHolder> {

    public interface OnRouteClickListener {
        void onRouteClick(Route route);
    }

    private final List<Route> routes;
    private OnRouteClickListener listener;

    public RouteAdapter(List<Route> routes) {
        this.routes = routes;
    }

    public void setOnRouteClickListener(OnRouteClickListener listener) {
        this.listener = listener;
    }

    @NonNull
    @Override
    public RouteViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View view = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.ly_trips_item, parent, false);
        return new RouteViewHolder(view);
    }

    @Override
    public void onBindViewHolder(@NonNull RouteViewHolder holder, int position) {
        Route route = routes.get(position);

        String fullName = route.getName();
        String[] parts = fullName.split(" - ", 2);
        holder.tvRouteName.setText(parts[0]);
        holder.tvRouteDesc.setText(parts.length > 1 ? parts[1] : "");
        holder.tvLabelVehicle.setText(route.getVehicleCount() + " vehiculos");

        if (route.isActive()) {
            holder.cvTextState.setText(R.string.txtActive);
            holder.cvTextState.setTextColor(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.green_dark));
            holder.cvColorState.setCardBackgroundColor(ColorStateList.valueOf(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.color_active)));

            holder.cvInfoState.setText(route.getVehicleCount() + " buses");
            holder.cvInfoState.setTextColor(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.green_dark));
            holder.cvState.setCardBackgroundColor(ColorStateList.valueOf(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.color_active)));
        } else {
            holder.cvTextState.setText(R.string.txtInactive);
            holder.cvTextState.setTextColor(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.red_dark));
            holder.cvColorState.setCardBackgroundColor(ColorStateList.valueOf(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.color_inactive)));

            holder.cvInfoState.setText("Sin buses");
            holder.cvInfoState.setTextColor(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.red_dark));
            holder.cvState.setCardBackgroundColor(ColorStateList.valueOf(
                    ContextCompat.getColor(holder.itemView.getContext(), R.color.color_inactive)));
        }

        holder.itemView.setOnClickListener(v -> {
            if (listener != null) listener.onRouteClick(route);
        });
    }

    @Override
    public int getItemCount() {
        return routes.size();
    }

    public static class RouteViewHolder extends RecyclerView.ViewHolder {

        TextView tvRouteName;
        TextView tvRouteDesc;
        TextView tvLabelVehicle;
        CardView cvColorState;
        TextView cvTextState;
        CardView cvState;
        TextView cvInfoState;

        public RouteViewHolder(@NonNull View itemView) {
            super(itemView);
            tvRouteName    = itemView.findViewById(R.id.tvRouteName);
            tvRouteDesc    = itemView.findViewById(R.id.tvRouteDesc);
            tvLabelVehicle = itemView.findViewById(R.id.tvLabelVehicle);
            cvColorState   = itemView.findViewById(R.id.cvColorState);
            cvTextState    = itemView.findViewById(R.id.cvTextState);
            cvState        = itemView.findViewById(R.id.cvState);
            cvInfoState    = itemView.findViewById(R.id.cvInfoState);
        }
    }
}
