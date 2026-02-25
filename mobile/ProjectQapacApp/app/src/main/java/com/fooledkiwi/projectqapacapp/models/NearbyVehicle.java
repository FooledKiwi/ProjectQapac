package com.fooledkiwi.projectqapacapp.models;

import com.google.gson.annotations.SerializedName;

public class NearbyVehicle {

    @SerializedName("id")
    private int id;

    @SerializedName("plate")
    private String plate;

    @SerializedName("route_name")
    private String routeName;

    @SerializedName("lat")
    private double lat;

    @SerializedName("lon")
    private double lon;

    public int getId() { return id; }
    public String getPlate() { return plate; }
    public String getRouteName() { return routeName; }
    public double getLat() { return lat; }
    public double getLon() { return lon; }
}
