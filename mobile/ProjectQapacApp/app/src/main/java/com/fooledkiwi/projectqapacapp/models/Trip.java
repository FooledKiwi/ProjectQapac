package com.fooledkiwi.projectqapacapp.models;

public class Trip {

    private String name;
    private String plate;
    private String duration;
    private float rating;
    private String driver;

    public Trip(String name, String plate, String duration, float rating, String driver) {
        this.name = name;
        this.plate = plate;
        this.duration = duration;
        this.rating = rating;
        this.driver = driver;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getPlate() {
        return plate;
    }

    public void setPlate(String plate) {
        this.plate = plate;
    }

    public String getDuration() {
        return duration;
    }

    public void setDuration(String duration) {
        this.duration = duration;
    }

    public float getRating() {
        return rating;
    }

    public void setRating(float rating) {
        this.rating = rating;
    }

    public String getDriver() {
        return driver;
    }

    public void setDriver(String driver) {
        this.driver = driver;
    }
}
