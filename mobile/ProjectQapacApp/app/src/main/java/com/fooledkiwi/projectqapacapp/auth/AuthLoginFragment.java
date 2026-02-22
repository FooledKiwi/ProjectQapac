package com.fooledkiwi.projectqapacapp.auth;

import android.content.Intent;
import android.os.Bundle;

import androidx.fragment.app.Fragment;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.EditText;

import com.fooledkiwi.projectqapacapp.MainActivity;
import com.fooledkiwi.projectqapacapp.R;

/**
 * A simple {@link Fragment} subclass.
 * Use the {@link AuthLoginFragment#newInstance} factory method to
 * create an instance of this fragment.
 */
public class AuthLoginFragment extends Fragment {


    public AuthLoginFragment() {
        // Required empty public constructor
    }

    /**
     * Use this factory method to create a new instance of
     * this fragment using the provided parameters.
     *
     * @param param1 Parameter 1.
     * @param param2 Parameter 2.
     * @return A new instance of fragment AuthLoginFragment.
     */
    // TODO: Rename and change types and number of parameters
    public static AuthLoginFragment newInstance(String param1, String param2) {
        AuthLoginFragment fragment = new AuthLoginFragment();
        Bundle args = new Bundle();
        // args.putString(ARG_PARAM1, param1);
        // args.putString(ARG_PARAM2, param2);
        fragment.setArguments(args);
        return fragment;
    }

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        View view = inflater.inflate(R.layout.fragment_auth_login, container, false);
        Button loginButton = view.findViewById(R.id.btn_loginConfirm);
        loginButton.setOnClickListener(v -> {
            Intent gotoMain = new Intent(view.getContext(), MainActivity.class);
            if(CheckLogin(v)) startActivity(gotoMain);
        });
        return view;
    }

    public boolean CheckLogin(View vw) {
        EditText inputUser = vw.findViewById(R.id.editText_loginUser);
        EditText inputPassword = vw.findViewById(R.id.editText_loginPassword);

        String username = inputUser.getText().toString().trim();
        String password = inputPassword.getText().toString();

        if (username.isEmpty()) {
            inputUser.setError(getString(R.string.error_empty_user));
            inputUser.requestFocus();
            return false;
        }

        if (password.isEmpty()) {
            inputPassword.setError(getString(R.string.error_empty_password));
            inputPassword.requestFocus();
            return false;
        }

        if (password.length() < 4) {
            inputPassword.setError(getString(R.string.error_short_password));
            inputPassword.requestFocus();
            return false;
        }


        // Ejemplo de lo que deberías llamar aquí (usando tu SqlLitePlates u otro Helper):
        // boolean loginExitoso = miDatabaseHelper.validarUsuario(username, password);
        //
        // if (loginExitoso) {
        //     Toast.makeText(getContext(), "Bienvenido", Toast.LENGTH_SHORT).show();
        //     // Navegar a la siguiente pantalla...
        // } else {
        //     Toast.makeText(getContext(), "Credenciales incorrectas", Toast.LENGTH_SHORT).show();
        // }
        return  true;
    }
}