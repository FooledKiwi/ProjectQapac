package com.fooledkiwi.projectqapacapp.models;

import com.google.gson.annotations.SerializedName;

public class UserDto {

    @SerializedName("id")
    private int id;

    @SerializedName("username")
    private String username;

    @SerializedName("full_name")
    private String fullName;

    @SerializedName("role")
    private String role;

    public int getId() { return id; }
    public String getUsername() { return username; }
    public String getFullName() { return fullName; }
    public String getRole() { return role; }
}
