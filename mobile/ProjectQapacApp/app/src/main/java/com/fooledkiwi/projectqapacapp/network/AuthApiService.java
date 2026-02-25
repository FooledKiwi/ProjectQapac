package com.fooledkiwi.projectqapacapp.network;

import com.fooledkiwi.projectqapacapp.models.LoginRequest;
import com.fooledkiwi.projectqapacapp.models.LoginResponse;

import retrofit2.Call;
import retrofit2.http.Body;
import retrofit2.http.POST;

public interface AuthApiService {
    @POST("api/v1/auth/login")
    Call<LoginResponse> login(@Body LoginRequest request);
}
