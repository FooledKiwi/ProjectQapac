package com.fooledkiwi.projectqapacapp.network;

import com.fooledkiwi.projectqapacapp.models.Stop;

import java.util.List;

import retrofit2.Call;
import retrofit2.http.GET;
import retrofit2.http.Query;

public interface StopsApiService {

    @GET("api/v1/stops/nearby")
    Call<List<Stop>> getNearbyStops(
            @Query("lat") double lat,
            @Query("lon") double lon,
            @Query("radius") double radius
    );
}
