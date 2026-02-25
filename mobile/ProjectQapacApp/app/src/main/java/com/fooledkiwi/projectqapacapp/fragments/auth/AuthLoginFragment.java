package com.fooledkiwi.projectqapacapp.fragments.auth;

import android.content.Intent;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.EditText;
import android.widget.ProgressBar;
import android.widget.Toast;

import androidx.fragment.app.Fragment;

import com.fooledkiwi.projectqapacapp.R;
import com.fooledkiwi.projectqapacapp.activities.MainActivity;
import com.fooledkiwi.projectqapacapp.models.LoginRequest;
import com.fooledkiwi.projectqapacapp.models.LoginResponse;
import com.fooledkiwi.projectqapacapp.network.ApiClient;
import com.fooledkiwi.projectqapacapp.session.SessionManager;

import retrofit2.Call;
import retrofit2.Callback;
import retrofit2.Response;

public class AuthLoginFragment extends Fragment {

    public AuthLoginFragment() {
        // Required empty public constructor
    }

    public static AuthLoginFragment newInstance() {
        return new AuthLoginFragment();
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_auth_login, container, false);

        Button loginButton = view.findViewById(R.id.btn_loginConfirm);
        loginButton.setOnClickListener(v -> attemptLogin(view));
        return view;
    }

    private void attemptLogin(View view) {
        EditText inputUser = view.findViewById(R.id.editText_loginUser);
        EditText inputPassword = view.findViewById(R.id.editText_loginPassword);
        Button loginButton = view.findViewById(R.id.btn_loginConfirm);
        ProgressBar progressBar = view.findViewById(R.id.pb_loginLoading);

        String username = inputUser.getText().toString().trim();
        String password = inputPassword.getText().toString();

        // Validaciones locales
        if (username.isEmpty()) {
            inputUser.setError(getString(R.string.error_empty_user));
            inputUser.requestFocus();
            return;
        }

        if (password.isEmpty()) {
            inputPassword.setError(getString(R.string.error_empty_password));
            inputPassword.requestFocus();
            return;
        }

        if (password.length() < 4) {
            inputPassword.setError(getString(R.string.error_short_password));
            inputPassword.requestFocus();
            return;
        }

        setLoading(loginButton, progressBar, true);
        Intent intent = new Intent(requireContext(), MainActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);

        LoginRequest request = new LoginRequest(username, password);
        ApiClient.getAuthService().login(request).enqueue(new Callback<LoginResponse>() {
            @Override
            public void onResponse(Call<LoginResponse> call, Response<LoginResponse> response) {
                if (!isAdded()) return;
                setLoading(loginButton, progressBar, false);
                if (response.isSuccessful() && response.body() != null) {
                    LoginResponse body = response.body();

                    // Guardar sesión
                    SessionManager session = new SessionManager(requireContext());
                    session.saveSession(body.getAccessToken(), body.getRefreshToken(), body.getUser());

                    // Navegar a MainActivity
                    Intent intent = new Intent(requireContext(), MainActivity.class);
                    intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TASK);
                    startActivity(intent);

                } else if (response.code() == 401) {
                    Toast.makeText(requireContext(),
                            getString(R.string.error_invalid_credentials),
                            Toast.LENGTH_LONG).show();
                } else {
                    Toast.makeText(requireContext(),
                            getString(R.string.error_server),
                            Toast.LENGTH_LONG).show();
                }
            }

            @Override
            public void onFailure(Call<LoginResponse> call, Throwable t) {
                if (!isAdded()) return;
                setLoading(loginButton, progressBar, false);
                Toast.makeText(requireContext(),
                        getString(R.string.error_network),
                        Toast.LENGTH_LONG).show();
            }
        });
    }

    private void setLoading(Button button, ProgressBar progressBar, boolean isLoading) {
        button.setEnabled(!isLoading);
        progressBar.setVisibility(isLoading ? View.VISIBLE : View.GONE);
    }
}
