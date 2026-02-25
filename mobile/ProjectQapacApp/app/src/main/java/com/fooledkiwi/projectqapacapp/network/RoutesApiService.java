package com.fooledkiwi.projectqapacapp.network;

import com.fooledkiwi.projectqapacapp.models.Route;
import com.fooledkiwi.projectqapacapp.models.RouteDetail;

import java.util.List;

import retrofit2.Call;
import retrofit2.http.GET;
import retrofit2.http.Path;

public interface RoutesApiService {

    @GET("api/v1/routes")
    Call<List<Route>> getRoutes();

    @GET("api/v1/routes/{id}")
    Call<RouteDetail> getRouteById(@Path("id") int id);
}
