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

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.models.Stop;
import com.fooledkiwi.projectqapacapp.network.ApiClient;
import com.fooledkiwi.projectqapacapp.network.StopsApiService;
import com.google.android.gms.location.FusedLocationProviderClient;
import com.google.android.gms.location.LocationServices;
import com.google.android.gms.maps.CameraUpdateFactory;
import com.google.android.gms.maps.GoogleMap;
import com.google.android.gms.maps.OnMapReadyCallback;
import com.google.android.gms.maps.SupportMapFragment;
import com.google.android.gms.maps.model.BitmapDescriptor;
import com.google.android.gms.maps.model.BitmapDescriptorFactory;
import com.google.android.gms.maps.model.LatLng;
import com.google.android.gms.maps.model.Marker;
import com.google.android.gms.maps.model.MarkerOptions;
import com.google.android.gms.maps.model.PolylineOptions;
import com.google.android.material.floatingactionbutton.FloatingActionButton;

import android.graphics.Color;
import android.location.Address;
import android.location.Geocoder;
import android.widget.Toast;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

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
    private TextView tvCurrentLocation;
    private ActivityResultLauncher<String[]> requestPermissionLauncher;

    private final List<Marker> stopMarkers = new ArrayList<>();

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
        tvCurrentLocation  = view.findViewById(R.id.tvCurrentLocationExplorer);

        layoutNoRoute.setVisibility(View.VISIBLE);
        layoutStopInfo.setVisibility(View.GONE);

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

        //FloatingActionButton fabSearch = view.findViewById(R.id.fabSearch);
        //fabSearch.setOnClickListener(v -> fetchNearbyStops());

        fusedLocationClient = LocationServices.getFusedLocationProviderClient(requireActivity());
        SupportMapFragment mapFragment = (SupportMapFragment) getChildFragmentManager()
            .findFragmentById(R.id.map_container);

        if (mapFragment != null) {
            mapFragment.getMapAsync(this);
        }
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

        //Todo esto es para probar
        Stop test = new Stop(0, "Parada insana", -7.165005036051442f, -78.49572883907857f);
        Stop t2 = new Stop(1, "Cruce o y q se", -7.165879291507253f, -78.50892986119936f);

        List<Stop> stops = new ArrayList<>();
        stops.add(test);
        stops.add(t2);
        stops.forEach(this::addStopMarker);

        map.addPolyline(new PolylineOptions()
                .add(new LatLng(test.getLat(), test.getLon()))
                .add(new LatLng(t2.getLat(), t2.getLon()))
                .width(10f)
                .color(Color.BLUE)
                .clickable(true));

        map.setOnPolylineClickListener(polyline ->
                Toast.makeText(requireContext(), "click en ruta", Toast.LENGTH_SHORT).show()
        );
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

        tvRouteName.setText(stop.getName());
        tvLabelVehicle.setText(stop.getLat() + ", " + stop.getLon());

        layoutNoRoute.setVisibility(View.GONE);
        layoutStopInfo.setVisibility(View.VISIBLE);
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
            // Geocoder no disponible o sin red — se deja el texto por defecto "[Ubicación]"
        }
    }
}
