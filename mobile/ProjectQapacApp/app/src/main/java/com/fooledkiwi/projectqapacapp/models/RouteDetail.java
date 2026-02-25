package com.fooledkiwi.projectqapacapp.models;

import com.google.gson.annotations.SerializedName;

import java.util.List;

public class RouteDetail {

    @SerializedName("id")
    private int id;

    @SerializedName("name")
    private String name;

    @SerializedName("active")
    private boolean active;

    @SerializedName("shape_polyline")
    private String shapePolyline;

    @SerializedName("stops")
    private List<RouteStop> stops;

    @SerializedName("vehicles")
    private List<RouteVehicle> vehicles;

    // --- Getters ---

    public int getId() { return id; }
    public String getName() { return name; }
    public boolean isActive() { return active; }
    public String getShapePolyline() { return shapePolyline; }
    public List<RouteStop> getStops() { return stops; }
    public List<RouteVehicle> getVehicles() { return vehicles; }

    // --- Nested: RouteStop ---

    public static class RouteStop {
        @SerializedName("id")
        private long id;

        @SerializedName("name")
        private String name;

        @SerializedName("lat")
        private double lat;

        @SerializedName("lon")
        private double lon;

        @SerializedName("sequence")
        private int sequence;

        public long getId() { return id; }
        public String getName() { return name; }
        public double getLat() { return lat; }
        public double getLon() { return lon; }
        public int getSequence() { return sequence; }
    }

    // --- Nested: RouteVehicle ---

    public static class RouteVehicle {
        @SerializedName("id")
        private int id;

        @SerializedName("plate")
        private String plate;

        @SerializedName("driver")
        private String driver;

        @SerializedName("collector")
        private String collector;

        @SerializedName("status")
        private String status;

        public int getId() { return id; }
        public String getPlate() { return plate; }
        public String getDriver() { return driver; }
        public String getCollector() { return collector; }
        public String getStatus() { return status; }
    }
}
