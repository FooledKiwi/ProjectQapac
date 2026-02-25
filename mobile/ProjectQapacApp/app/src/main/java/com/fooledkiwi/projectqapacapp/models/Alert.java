package com.fooledkiwi.projectqapacapp.models;

public class Alert {

    private final String title;
    private final String subtitle;
    private final String time;

    public Alert(String title, String subtitle, String time) {
        this.title    = title;
        this.subtitle = subtitle;
        this.time     = time;
    }

    public String getTitle()    { return title; }
    public String getSubtitle() { return subtitle; }
    public String getTime()     { return time; }
}
