package com.fooledkiwi.projectqapacapp.models;

import com.google.gson.annotations.SerializedName;

public class Stop {

    @SerializedName("id")
    private long id;

    @SerializedName("name")
    private String name;

    @SerializedName("lat")
    private double lat;

    @SerializedName("lon")
    private double lon;

    @SerializedName("eta_seconds")
    private int etaSeconds;

    public Stop(long id, String name, double lat, double lon) {
        this.id = id;
        this.name = name;
        this.lat = lat;
        this.lon = lon;
    }

    public long getId() {
        return id;
    }

    public void setId(long id) {
        this.id = id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public double getLat() {
        return lat;
    }

    public void setLat(double lat) {
        this.lat = lat;
    }

    public double getLon() {
        return lon;
    }

    public void setLon(double lon) {
        this.lon = lon;
    }

    public int getEtaSeconds() {
        return etaSeconds;
    }

    public void setEtaSeconds(int etaSeconds) {
        this.etaSeconds = etaSeconds;
    }
}
