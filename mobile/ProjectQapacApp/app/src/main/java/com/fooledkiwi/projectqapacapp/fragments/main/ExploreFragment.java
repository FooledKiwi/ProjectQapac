package com.fooledkiwi.projectqapacapp.fragments.main;

import android.Manifest;
import android.content.Context;
import android.content.pm.PackageManager;
import android.graphics.Bitmap;
import android.graphics.Canvas;
import android.graphics.drawable.Drawable;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.TextView;

import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.core.content.ContextCompat;
import androidx.fragment.app.Fragment;
import androidx.recyclerview.widget.LinearLayoutManager;
import androidx.recyclerview.widget.RecyclerView;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.adapters.RouteAdapter;
import com.fooledkiwi.projectqapacapp.models.NearbyVehicle;
import com.fooledkiwi.projectqapacapp.models.Route;
import com.fooledkiwi.projectqapacapp.models.RouteDetail;
import com.fooledkiwi.projectqapacapp.models.Stop;
import com.fooledkiwi.projectqapacapp.network.ApiClient;
import com.fooledkiwi.projectqapacapp.network.StopsApiService;
import com.fooledkiwi.projectqapacapp.session.SessionManager;
import com.google.android.gms.location.FusedLocationProviderClient;
import com.google.android.gms.location.LocationServices;
import com.google.android.gms.maps.CameraUpdateFactory;
import com.google.android.gms.maps.GoogleMap;
import com.google.android.gms.maps.OnMapReadyCallback;
import com.google.android.gms.maps.SupportMapFragment;
import com.google.android.gms.maps.model.BitmapDescriptor;
import com.google.android.gms.maps.model.BitmapDescriptorFactory;
import com.google.android.gms.maps.model.LatLng;
import com.google.android.gms.maps.model.LatLngBounds;
import com.google.android.gms.maps.model.Marker;
import com.google.android.gms.maps.model.MarkerOptions;
import com.google.android.gms.maps.model.Polyline;
import com.google.android.gms.maps.model.PolylineOptions;
import com.google.android.material.bottomsheet.BottomSheetBehavior;
import com.google.android.material.floatingactionbutton.FloatingActionButton;

import android.graphics.Color;
import android.location.Address;
import android.location.Geocoder;
import android.widget.Toast;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

import android.os.Handler;
import android.os.Looper;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;

public class ExploreFragment extends Fragment implements OnMapReadyCallback {

    private GoogleMap map;
    private FusedLocationProviderClient fusedLocationClient;
    private View layoutNoRoute;
    private View layoutStopInfo;
    private View layoutNoPermission;
    private TextView tvRouteName;
    private TextView tvLabelVehicle;
    private TextView tvEtaSeconds;
    private TextView tvCurrentLocation;
    private ActivityResultLauncher<String[]> requestPermissionLauncher;

    private final List<Marker> stopMarkers = new ArrayList<>();
    private final List<Marker> vehicleMarkers = new ArrayList<>();

    private RecyclerView rvRoutes;
    private RouteAdapter routeAdapter;
    private final List<Route> routeList = new ArrayList<>();
    private final Map<Integer, RouteDetail> routeDetailCache = new HashMap<>();
    private Polyline currentRoutePolyline = null;
    private BottomSheetBehavior<View> bottomSheetBehavior;

    private SessionManager sessionManager;
    private final Handler vehiclePollingHandler = new Handler(Looper.getMainLooper());
    private Runnable vehiclePollingRunnable;
    private static final long VEHICLE_POLL_INTERVAL_MS = 10_000L;

    public ExploreFragment() {
        // Required empty public constructor
    }

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        requestPermissionLauncher = registerForActivityResult(
            new ActivityResultContracts.RequestMultiplePermissions(), result -> {
                Boolean fineGranted = result.getOrDefault(Manifest.permission.ACCESS_FINE_LOCATION, false);
                if (fineGranted != null && fineGranted) {
                    layoutNoPermission.setVisibility(View.GONE);
                    setMapGesturesEnabled(true);
                    enableMyLocation();
                }
            });
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        return inflater.inflate(R.layout.fragment_explore, container, false);
    }

    @Override
    public void onViewCreated(@NonNull View view, @Nullable Bundle savedInstanceState) {
        super.onViewCreated(view, savedInstanceState);

        layoutNoRoute      = view.findViewById(R.id.layoutNoRoute);
        layoutStopInfo     = view.findViewById(R.id.layoutStopInfo);
        layoutNoPermission = view.findViewById(R.id.layoutNoPermission);
        tvRouteName        = view.findViewById(R.id.tvRouteName);
        tvLabelVehicle     = view.findViewById(R.id.tvLabelVehicle);
        tvEtaSeconds       = view.findViewById(R.id.tvEtaSeconds);
        tvCurrentLocation  = view.findViewById(R.id.tvCurrentLocationExplorer);

        layoutNoRoute.setVisibility(View.VISIBLE);
        layoutStopInfo.setVisibility(View.GONE);

        bottomSheetBehavior = BottomSheetBehavior.from(view.findViewById(R.id.ll_bottom_sheet));

        if (ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION)
                != PackageManager.PERMISSION_GRANTED) {
            layoutNoPermission.setVisibility(View.VISIBLE);
        } else {
            layoutNoPermission.setVisibility(View.GONE);
        }

        Button btnRequestPermission = view.findViewById(R.id.btnRequestPermission);
        btnRequestPermission.setOnClickListener(v ->
            requestPermissionLauncher.launch(new String[]{
                Manifest.permission.ACCESS_FINE_LOCATION,
                Manifest.permission.ACCESS_COARSE_LOCATION
            })
        );

        rvRoutes = view.findViewById(R.id.rvRoutes);
        routeAdapter = new RouteAdapter(routeList);
        rvRoutes.setLayoutManager(new LinearLayoutManager(requireContext()));
        rvRoutes.setAdapter(routeAdapter);
        routeAdapter.setOnRouteClickListener(route -> {
            bottomSheetBehavior.setState(BottomSheetBehavior.STATE_COLLAPSED);
            drawRoutePolylineById(route.getId());
        });

        FloatingActionButton fabSearch = view.findViewById(R.id.fabSearch);
        fabSearch.setOnClickListener(v -> {
            fetchNearbyStops();
            fetchNearbyVehicles();
        });

        fusedLocationClient = LocationServices.getFusedLocationProviderClient(requireActivity());
        SupportMapFragment mapFragment = (SupportMapFragment) getChildFragmentManager()
            .findFragmentById(R.id.map_container);

        if (mapFragment != null) {
            mapFragment.getMapAsync(this);
        }

        fetchNearbyStops();
        fetchRoutes();

        sessionManager = new SessionManager(requireContext());
        startVehiclePolling();
    }

    @Override
    public void onMapReady(@NonNull GoogleMap googleMap) {
        map = googleMap;

        boolean hasPermission = ContextCompat.checkSelfPermission(requireContext(),
                Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED;
        setMapGesturesEnabled(hasPermission);

        enableMyLocation();

        map.setOnMarkerClickListener(marker -> {
            onStopMarkerClick(marker);
            return true;
        });
    }

    @Override
    public void onResume() {
        super.onResume();
        if (sessionManager != null && !sessionManager.isLoggedIn()) {
            startVehiclePolling();
        }
    }

    @Override
    public void onPause() {
        super.onPause();
        stopVehiclePolling();
    }

    private void startVehiclePolling() {
        // Avoid scheduling duplicates
        stopVehiclePolling();
        vehiclePollingRunnable = new Runnable() {
            @Override
            public void run() {
                fetchNearbyVehicles();
                vehiclePollingHandler.postDelayed(this, VEHICLE_POLL_INTERVAL_MS);
            }
        };
        vehiclePollingHandler.post(vehiclePollingRunnable);
    }

    private void stopVehiclePolling() {
        if (vehiclePollingRunnable != null) {
            vehiclePollingHandler.removeCallbacks(vehiclePollingRunnable);
            vehiclePollingRunnable = null;
        }
    }

    private void fetchNearbyVehicles() {
        if (!isAdded()) return;
        if (ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION)
                != PackageManager.PERMISSION_GRANTED) return;

        fusedLocationClient.getLastLocation().addOnSuccessListener(requireActivity(), location -> {
            if (location == null || !isAdded()) return;
            double lat = location.getLatitude();
            double lon = location.getLongitude();

            ApiClient.getVehiclesService()
                    .getNearbyVehicles(lat, lon, 5000f)
                    .enqueue(new Callback<List<NearbyVehicle>>() {
                        @Override
                        public void onResponse(@NonNull Call<List<NearbyVehicle>> call,
                                               @NonNull Response<List<NearbyVehicle>> response) {
                            if (!isAdded()) return;
                            if (response.isSuccessful() && response.body() != null) {
                                clearVehicleMarkers();
                                for (NearbyVehicle vehicle : response.body()) {
                                    addVehicleMarker(vehicle);
                                }
                            }
                        }

                        @Override
                        public void onFailure(@NonNull Call<List<NearbyVehicle>> call,
                                              @NonNull Throwable t) {
                            // Fallo silencioso: se reintentara en el proximo ciclo
                        }
                    });
        });
    }

    private void clearVehicleMarkers() {
        for (Marker marker : vehicleMarkers) {
            marker.remove();
        }
        vehicleMarkers.clear();
    }

    private void addVehicleMarker(NearbyVehicle vehicle) {
        if (map == null) return;
        LatLng position = new LatLng(vehicle.getLat(), vehicle.getLon());
        Marker marker = map.addMarker(new MarkerOptions()
                .position(position)
                .title(vehicle.getPlate())
                .snippet(vehicle.getRouteName())
                .icon(customIcon(requireContext(), R.drawable.icon_bus_blue)));
        if (marker != null) {
            // Tag null intencionalmente: el listener de marcadores ignora vehiculos
            vehicleMarkers.add(marker);
        }
    }

    private void fetchNearbyStops() {
        if (ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION)
                != PackageManager.PERMISSION_GRANTED) {
            Toast.makeText(requireContext(), "Se necesita permiso de ubicacion", Toast.LENGTH_SHORT).show();
            return;
        }

        fusedLocationClient.getLastLocation().addOnSuccessListener(requireActivity(), location -> {
            if (location == null) {
                Toast.makeText(requireContext(), "No se pudo obtener la ubicacion", Toast.LENGTH_SHORT).show();
                return;
            }

            double lat = location.getLatitude();
            double lon = location.getLongitude();

            StopsApiService service = ApiClient.getStopsService();
            service.getNearbyStops(lat, lon, 1000).enqueue(new Callback<List<Stop>>() {
                @Override
                public void onResponse(@NonNull Call<List<Stop>> call, @NonNull Response<List<Stop>> response) {
                    if (!isAdded()) return;
                    if (response.isSuccessful() && response.body() != null) {
                        clearStopMarkers();
                        for (Stop stop : response.body()) {
                            addStopMarker(stop);
                        }
                        int count = response.body().size();
                        Toast.makeText(requireContext(),
                                count + " parada(s) encontrada(s)",
                                Toast.LENGTH_SHORT).show();
                    } else {
                        Toast.makeText(requireContext(),
                                "Error al obtener paradas: " + response.code(),
                                Toast.LENGTH_SHORT).show();
                    }
                }

                @Override
                public void onFailure(@NonNull Call<List<Stop>> call, @NonNull Throwable t) {
                    if (!isAdded()) return;
                    Toast.makeText(requireContext(),
                            "Error de red: " + t.getMessage(),
                            Toast.LENGTH_SHORT).show();
                }
            });
        });
    }

    private void fetchRoutes() {
        ApiClient.getRoutesService().getRoutes().enqueue(new Callback<List<Route>>() {
            @Override
            public void onResponse(@NonNull Call<List<Route>> call,
                                   @NonNull Response<List<Route>> response) {
                if (!isAdded()) return;
                if (response.isSuccessful() && response.body() != null) {
                    routeList.clear();
                    routeList.addAll(response.body());
                    routeAdapter.notifyDataSetChanged();
                    // Eagerly cache route details for polyline/stop lookup
                    for (Route route : routeList) {
                        prefetchRouteDetail(route.getId());
                    }
                }
            }

            @Override
            public void onFailure(@NonNull Call<List<Route>> call, @NonNull Throwable t) {
                // Error de red silencioso: las rutas no son criticas para el flujo principal
            }
        });
    }

    private void prefetchRouteDetail(int routeId) {
        if (routeDetailCache.containsKey(routeId)) return;
        ApiClient.getRoutesService().getRouteById(routeId).enqueue(new Callback<RouteDetail>() {
            @Override
            public void onResponse(@NonNull Call<RouteDetail> call,
                                   @NonNull Response<RouteDetail> response) {
                if (!isAdded()) return;
                if (response.isSuccessful() && response.body() != null) {
                    routeDetailCache.put(routeId, response.body());
                }
            }

            @Override
            public void onFailure(@NonNull Call<RouteDetail> call, @NonNull Throwable t) {
                // Fallo silencioso: polyline no estara disponible para esta ruta
            }
        });
    }

    private void clearStopMarkers() {
        for (Marker marker : stopMarkers) {
            marker.remove();
        }
        stopMarkers.clear();
    }

    private void setMapGesturesEnabled(boolean enabled) {
        if (map == null) return;
        map.getUiSettings().setScrollGesturesEnabled(enabled);
        map.getUiSettings().setZoomGesturesEnabled(enabled);
        map.getUiSettings().setTiltGesturesEnabled(enabled);
        map.getUiSettings().setRotateGesturesEnabled(enabled);
        map.getUiSettings().setZoomControlsEnabled(enabled);
    }

    private void enableMyLocation() {
        if (map == null) return;
        if (ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.ACCESS_FINE_LOCATION)
                == PackageManager.PERMISSION_GRANTED) {
            map.setMyLocationEnabled(true);
            fusedLocationClient.getLastLocation().addOnSuccessListener(requireActivity(), location -> {
                if (location != null) {
                    LatLng miUbicacion = new LatLng(location.getLatitude(), location.getLongitude());
                    map.animateCamera(CameraUpdateFactory.newLatLngZoom(miUbicacion, 15f));
                    updateCurrentLocationLabel(location.getLatitude(), location.getLongitude());
                }
            });
        }
    }

    private BitmapDescriptor customIcon(Context context, int vectorResId) {
        int width = 100;
        int height = 100;
        Drawable vectorDrawable = ContextCompat.getDrawable(context, vectorResId);
        vectorDrawable.setBounds(0, 0, width, height);
        Bitmap bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888);
        Canvas canvas = new Canvas(bitmap);
        vectorDrawable.draw(canvas);
        return BitmapDescriptorFactory.fromBitmap(bitmap);
    }

    private void addStopMarker(Stop stop) {
        if (map == null) return;
        LatLng position = new LatLng(stop.getLat(), stop.getLon());
        Marker marker = map.addMarker(new MarkerOptions()
                .position(position)
                .title(stop.getName())
                .icon(customIcon(requireContext(), R.drawable.icon_geo_dark)));
        if (marker != null) {
            marker.setTag(stop);
            stopMarkers.add(marker);
        }
    }

    private void onStopMarkerClick(Marker marker) {
        Stop stop = (Stop) marker.getTag();
        if (stop == null) return;

        ApiClient.getStopsService().getStopById(stop.getId()).enqueue(new Callback<Stop>() {
            @Override
            public void onResponse(@NonNull Call<Stop> call, @NonNull Response<Stop> response) {
                if (!isAdded()) return;
                if (response.isSuccessful() && response.body() != null) {
                    Stop detail = response.body();
                    tvRouteName.setText(detail.getName());
                    tvLabelVehicle.setText(detail.getLat() + ", " + detail.getLon());
                    tvEtaSeconds.setText(formatEta(detail.getEtaSeconds()));
                    layoutNoRoute.setVisibility(View.GONE);
                    layoutStopInfo.setVisibility(View.VISIBLE);

                } else {
                    Toast.makeText(requireContext(),
                            "Error al obtener detalle: " + response.code(),
                            Toast.LENGTH_SHORT).show();
                }
            }

            @Override
            public void onFailure(@NonNull Call<Stop> call, @NonNull Throwable t) {
                if (!isAdded()) return;
                Toast.makeText(requireContext(),
                        "Error de red: " + t.getMessage(),
                        Toast.LENGTH_SHORT).show();
            }
        });
    }

    private void drawRoutePolylineById(int routeId) {
        RouteDetail cached = routeDetailCache.get(routeId);
        if (cached != null) {
            applyRoutePolyline(cached);
            return;
        }
        ApiClient.getRoutesService().getRouteById(routeId).enqueue(new Callback<RouteDetail>() {
            @Override
            public void onResponse(@NonNull Call<RouteDetail> call,
                                   @NonNull Response<RouteDetail> response) {
                if (!isAdded()) return;
                if (response.isSuccessful() && response.body() != null) {
                    routeDetailCache.put(routeId, response.body());
                    applyRoutePolyline(response.body());
                } else {
                    Toast.makeText(requireContext(),
                            "Error al cargar ruta: " + response.code(),
                            Toast.LENGTH_SHORT).show();
                }
            }

            @Override
            public void onFailure(@NonNull Call<RouteDetail> call, @NonNull Throwable t) {
                if (!isAdded()) return;
                Toast.makeText(requireContext(),
                        "Error de red: " + t.getMessage(),
                        Toast.LENGTH_SHORT).show();
            }
        });
    }


    private void applyRoutePolyline(RouteDetail detail) {
        if (map == null || detail.getShapePolyline() == null) return;

        List<LatLng> points = parseWktLinestring(detail.getShapePolyline());
        if (points.isEmpty()) return;

        if (currentRoutePolyline != null) {
            currentRoutePolyline.remove();
            currentRoutePolyline = null;
        }

        currentRoutePolyline = map.addPolyline(new PolylineOptions()
                .addAll(points)
                .width(8f)
                .color(Color.parseColor("#FF6200EE"))
                .geodesic(true));

        LatLngBounds.Builder boundsBuilder = new LatLngBounds.Builder();
        for (LatLng point : points) {
            boundsBuilder.include(point);
        }
        map.animateCamera(CameraUpdateFactory.newLatLngBounds(boundsBuilder.build(), 80));
    }

    private void drawRoutePolylineForStop(long stopId) {
        if (map == null) return;
        RouteDetail matched = null;
        for (RouteDetail detail : routeDetailCache.values()) {
            if (detail.getStops() == null) continue;
            for (RouteDetail.RouteStop rs : detail.getStops()) {
                if (rs.getId() == stopId) {
                    matched = detail;
                    break;
                }
            }
            if (matched != null) break;
        }

        if (currentRoutePolyline != null) {
            currentRoutePolyline.remove();
            currentRoutePolyline = null;
        }

        if (matched != null) {
            applyRoutePolyline(matched);
        }
    }

    private List<LatLng> parseWktLinestring(String wkt) {
        List<LatLng> points = new ArrayList<>();
        try {
            // Strip prefix "LINESTRING(" and suffix ")"
            int start = wkt.indexOf('(');
            int end = wkt.lastIndexOf(')');
            if (start < 0 || end < 0 || end <= start) return points;
            String coords = wkt.substring(start + 1, end).trim();
            for (String pair : coords.split(",")) {
                String[] parts = pair.trim().split("\\s+");
                if (parts.length < 2) continue;
                double lon = Double.parseDouble(parts[0]);
                double lat = Double.parseDouble(parts[1]);
                points.add(new LatLng(lat, lon));
            }
        } catch (Exception e) {
            // Malformed WKT: return whatever was parsed so far
        }
        return points;
    }

    private String formatEta(int etaSeconds) {
        if (etaSeconds <= 0) return "--:--";
        int minutes = etaSeconds / 60;
        int seconds = etaSeconds % 60;
        return String.format(Locale.getDefault(), "%02d:%02d", minutes, seconds);
    }

    private void updateCurrentLocationLabel(double lat, double lon) {
        if (tvCurrentLocation == null || !isAdded()) return;
        try {
            Geocoder geocoder = new Geocoder(requireContext(), Locale.getDefault());
            List<Address> addresses = geocoder.getFromLocation(lat, lon, 1);
            if (addresses != null && !addresses.isEmpty()) {
                Address address = addresses.get(0);
                String city = address.getLocality();
                String postalCode = address.getPostalCode();
                if (city == null || city.isEmpty()) city = address.getSubAdminArea();
                if (city == null || city.isEmpty()) city = address.getAdminArea();
                if (city != null && !city.isEmpty()) {
                    String finalCity = city + ", " + postalCode;
                    requireActivity().runOnUiThread(() -> tvCurrentLocation.setText(finalCity));
                }
            }
        } catch (Exception e) {
            // Geocoder no disponible o sin red no hay internet pe
        }
    }
}
