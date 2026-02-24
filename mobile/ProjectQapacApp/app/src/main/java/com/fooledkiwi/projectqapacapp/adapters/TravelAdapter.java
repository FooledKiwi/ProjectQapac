package com.fooledkiwi.projectqapacapp.adapters;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.RatingBar;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.models.Travel;

import java.util.List;

public class TravelAdapter extends RecyclerView.Adapter<TravelAdapter.TravelViewHolder> {

    private final List<Travel> travels;

    public TravelAdapter(List<Travel> travels) {
        this.travels = travels;
    }

    @NonNull
    @Override
    public TravelViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View view = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.ly_rating_item, parent, false);
        return new TravelViewHolder(view);
    }

    @Override
    public void onBindViewHolder(@NonNull TravelViewHolder holder, int position) {
        Travel travel = travels.get(position);
        holder.tvRouteName.setText(travel.getName());
        holder.tvLabelVehicle.setText(travel.getPlate());
        holder.tvDurationTrip.setText(travel.getDuration());
        holder.tvDriverName.setText(travel.getDriver());
        holder.rbRatingBar.setRating(travel.getRating());
    }

    @Override
    public int getItemCount() {
        return travels.size();
    }

    public static class TravelViewHolder extends RecyclerView.ViewHolder {

        TextView tvRouteName;
        TextView tvLabelVehicle;
        TextView tvDurationTrip;
        TextView tvDriverName;
        RatingBar rbRatingBar;

        public TravelViewHolder(@NonNull View itemView) {
            super(itemView);
            tvRouteName    = itemView.findViewById(R.id.tvRouteName);
            tvLabelVehicle = itemView.findViewById(R.id.tvLabelVehicle);
            tvDurationTrip = itemView.findViewById(R.id.tvDurationTrip);
            tvDriverName   = itemView.findViewById(R.id.tvDriverName);
            rbRatingBar    = itemView.findViewById(R.id.rbRatingBar);
        }
    }
}
