package com.fooledkiwi.projectqapacapp.models;

import com.google.gson.annotations.SerializedName;

public class DriverPositionRequest {

    @SerializedName("lat")
    private double lat;

    @SerializedName("lon")
    private double lon;

    @SerializedName("heading")
    private Double heading;

    @SerializedName("speed")
    private Double speed;

    public DriverPositionRequest(double lat, double lon, Double heading, Double speed) {
        this.lat = lat;
        this.lon = lon;
        this.heading = heading;
        this.speed = speed;
    }

    public double getLat() { return lat; }
    public void setLat(double lat) { this.lat = lat; }

    public double getLon() { return lon; }
    public void setLon(double lon) { this.lon = lon; }

    public Double getHeading() { return heading; }
    public void setHeading(Double heading) { this.heading = heading; }

    public Double getSpeed() { return speed; }
    public void setSpeed(Double speed) { this.speed = speed; }
}
