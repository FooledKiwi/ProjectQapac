package com.fooledkiwi.projectqapacapp.services;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.Service;
import android.content.Intent;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.util.Log;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.core.app.NotificationCompat;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.activities.AuthActivity;
import com.fooledkiwi.projectqapacapp.models.DriverPositionRequest;
import com.fooledkiwi.projectqapacapp.network.ApiClient;
import com.fooledkiwi.projectqapacapp.session.SessionManager;
import com.google.android.gms.location.FusedLocationProviderClient;
import com.google.android.gms.location.LocationServices;
import com.google.android.gms.location.Priority;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;

public class LocationReporterService extends Service {
    private static final String TAG = "LocationReporter";
    private static final String CHANNEL_ID = "location_reporter";
    private static final int NOTIFICATION_ID = 1001;
    private static final long INTERVAL_MS = 10_000L;
    private Handler handler;
    private Runnable reportRunnable;
    private FusedLocationProviderClient fusedLocationClient;
    private SessionManager sessionManager;
    @Override
    public void onCreate() {
        super.onCreate();
        handler = new Handler(Looper.getMainLooper());
        fusedLocationClient = LocationServices.getFusedLocationProviderClient(this);
        sessionManager = new SessionManager(this);
        createNotificationChannel();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (!sessionManager.isLoggedIn()) {
            redirectToAuthAndStop();
            return START_NOT_STICKY;
        }

        startForeground(NOTIFICATION_ID, buildNotification());
        reportRunnable = new Runnable() {
            @Override
            public void run() {
                if (!sessionManager.isLoggedIn()) {
                    redirectToAuthAndStop();
                    return;
                }
                reportCurrentPosition();
                handler.postDelayed(this, INTERVAL_MS);
            }
        };

        handler.post(reportRunnable);
        return START_STICKY;
    }

    private void reportCurrentPosition() {
        try {
            fusedLocationClient
                    .getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, null)
                    .addOnSuccessListener(location -> {
                        if (location == null) {
                            Log.d(TAG, "Ubicacion no disponible, se reintentara en el siguiente ciclo");
                            return;
                        }

                        String token = "Bearer " + sessionManager.getAccessToken();
                        DriverPositionRequest body = new DriverPositionRequest(
                                location.getLatitude(),
                                location.getLongitude(),
                                location.hasBearing() ? (double) location.getBearing() : null,
                                location.hasSpeed()   ? (double) (location.getSpeed() * 3.6f) : null
                        );

                        ApiClient.getDriverService()
                                .reportPosition(token, body)
                                .enqueue(new Callback<Void>() {
                                    @Override
                                    public void onResponse(@NonNull Call<Void> call,
                                                           @NonNull Response<Void> response) {
                                        if (response.code() == 401 || response.code() == 403) {
                                            Log.w(TAG, "Token invalido (" + response.code() + "), redirigiendo a auth");
                                            redirectToAuthAndStop();
                                        } else {
                                            Log.d(TAG, "Posicion reportada: "
                                                    + location.getLatitude() + ", "
                                                    + location.getLongitude()
                                                    + " | HTTP " + response.code());
                                        }
                                    }

                                    @Override
                                    public void onFailure(@NonNull Call<Void> call,
                                                          @NonNull Throwable t) {
                                        Log.w(TAG, "Error de red al reportar posicion: " + t.getMessage());
                                    }
                                });
                    });
        } catch (SecurityException e) {
            Log.e(TAG, "Permiso de ubicacion no concedido: " + e.getMessage());
            stopSelf();
        }
    }

    private void redirectToAuthAndStop() {
        sessionManager.clearSession();
        Intent intent = new Intent(this, AuthActivity.class);
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);
        startActivity(intent);
        stopSelf();
    }

    @Override
    public void onDestroy() {
        super.onDestroy();
        if (handler != null && reportRunnable != null) {
            handler.removeCallbacks(reportRunnable);
        }
        Toast.makeText(this, "Se paró el servicio de reporte de ubicación.", Toast.LENGTH_LONG).show();
        Log.d(TAG, "Servicio detenido");
    }

    @Nullable
    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    private void createNotificationChannel() {
        NotificationChannel channel = new NotificationChannel(
                CHANNEL_ID,
                "Reporte de ubicacion",
                NotificationManager.IMPORTANCE_LOW
        );
        channel.setDescription("Notificacion activa mientras se reporta la ubicacion del conductor");
        channel.setSound(null, null);
        NotificationManager manager = getSystemService(NotificationManager.class);
        if (manager != null) {
            manager.createNotificationChannel(channel);
        }
    }

    private Notification buildNotification() {
        return new NotificationCompat.Builder(this, CHANNEL_ID)
                .setContentTitle("Qapac — Conductor activo")
                .setContentText("Reportando ubicacion en segundo plano")
                .setSmallIcon(R.drawable.icon_geo_dark)
                .setOngoing(true)
                .setSilent(true)
                .build();
    }
}
