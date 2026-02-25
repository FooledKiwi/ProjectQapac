package com.fooledkiwi.projectqapacapp.network;

import com.fooledkiwi.projectqapacapp.models.DriverPositionRequest;

import retrofit2.Call;
import retrofit2.http.Body;
import retrofit2.http.Header;
import retrofit2.http.POST;

public interface DriverApiService {

    @POST("api/v1/driver/position")
    Call<Void> reportPosition(
            @Header("Authorization") String bearerToken,
            @Body DriverPositionRequest body
    );
}
