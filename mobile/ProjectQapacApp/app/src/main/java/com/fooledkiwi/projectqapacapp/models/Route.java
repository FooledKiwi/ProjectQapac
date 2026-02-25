package com.fooledkiwi.projectqapacapp.models;

import com.google.gson.annotations.SerializedName;

public class Route {

    @SerializedName("id")
    private int id;

    @SerializedName("name")
    private String name;

    @SerializedName("active")
    private boolean active;

    @SerializedName("vehicle_count")
    private int vehicleCount;

    public Route(int id, String name, boolean active, int vehicleCount) {
        this.id = id;
        this.name = name;
        this.active = active;
        this.vehicleCount = vehicleCount;
    }

    public int getId() { return id; }
    public void setId(int id) { this.id = id; }

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }

    public boolean isActive() { return active; }
    public void setActive(boolean active) { this.active = active; }

    public int getVehicleCount() { return vehicleCount; }
    public void setVehicleCount(int vehicleCount) { this.vehicleCount = vehicleCount; }
}
