package com.fooledkiwi.projectqapacapp.network;

import com.fooledkiwi.projectqapacapp.models.NearbyVehicle;

import java.util.List;

import retrofit2.Call;
import retrofit2.http.GET;
import retrofit2.http.Query;

public interface VehiclesApiService {

    @GET("api/v1/vehicles/nearby")
    Call<List<NearbyVehicle>> getNearbyVehicles(
            @Query("lat") double lat,
            @Query("lon") double lon,
            @Query("radius") float radius
    );
}
