package com.fooledkiwi.projectqapacapp.adapters;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.RatingBar;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.models.Trip;

import java.util.List;

public class TripAdapter extends RecyclerView.Adapter<TripAdapter.TravelViewHolder> {

    private final List<Trip> trips;

    public TripAdapter(List<Trip> trips) {
        this.trips = trips;
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
        Trip trip = trips.get(position);
        holder.tvRouteName.setText(trip.getName());
        holder.tvLabelVehicle.setText(trip.getPlate());
        holder.tvDurationTrip.setText(trip.getDuration());
        holder.tvDriverName.setText(trip.getDriver());
        holder.rbRatingBar.setRating(trip.getRating());
    }

    @Override
    public int getItemCount() {
        return trips.size();
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
