package com.fooledkiwi.projectqapacapp.auth;

import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.Bundle;

import androidx.fragment.app.Fragment;

import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.Button;
import android.widget.EditText;
import android.widget.Spinner;
import android.widget.Toast;

import com.fooledkiwi.projectqapacapp.MainActivity;
import com.fooledkiwi.projectqapacapp.R;

/**
 * A simple {@link Fragment} subclass.
 * Use the {@link AuthRegisterFragment#newInstance} factory method to
 * create an instance of this fragment.
 */
public class AuthRegisterFragment extends Fragment {

    public AuthRegisterFragment() {
        // Required empty public constructor
    }

    // TODO: Rename and change types and number of parameters
    public static AuthRegisterFragment newInstance(String param1, String param2) {
        AuthRegisterFragment fragment = new AuthRegisterFragment();
        Bundle args = new Bundle();
        //args.putString(ARG_PARAM1, param1);
        //args.putString(ARG_PARAM2, param2);
        fragment.setArguments(args);
        return fragment;
    }

    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        /*
        if (getArguments() != null) {
            mParam1 = getArguments().getString(ARG_PARAM1);
            mParam2 = getArguments().getString(ARG_PARAM2);
        }*/
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        // Inflate the layout for this fragment
        View vw = inflater.inflate(R.layout.fragment_auth_register, container, false);
        Spinner userType = vw.findViewById(R.id.sp_userType);
        String[] types = {"Selecciona para que usas la app","Soy conductor", "Soy Usuario"};
        ArrayAdapter<String> adapter = new ArrayAdapter<>(vw.getContext(), android.R.layout.simple_spinner_dropdown_item, types);
        userType.setAdapter(adapter);

        Button loginButton = vw.findViewById(R.id.btn_registerConfirm);
        loginButton.setOnClickListener(v -> {
            Intent gotoMain = new Intent(vw.getContext(), MainActivity.class);
            if(CheckRegister()) startActivity(gotoMain);
        });
        return vw;
    }

    public boolean CheckRegister() {
        View root = requireView();

        EditText inputUser = root.findViewById(R.id.editText_registerUser);
        EditText inputPassword = root.findViewById(R.id.editText_registerPassword);
        EditText inputConfirm = root.findViewById(R.id.editText_registerConfirm);
        EditText inputPhone = root.findViewById(R.id.editTextPhone);
        Spinner spUserType = root.findViewById(R.id.sp_userType);

        String username = inputUser.getText().toString().trim();
        String password = inputPassword.getText().toString();
        String confirmPass = inputConfirm.getText().toString();
        String phone = inputPhone.getText().toString().trim();

        if (username.isEmpty()) {
            inputUser.setError(getString(R.string.error_empty_user));
            inputUser.requestFocus();
            return false;
        }

        if (password.length() < 6) {
            inputPassword.setError(getString(R.string.error_short_password));
            inputPassword.requestFocus();
            return false;
        }

        if (!confirmPass.equals(password)) {
            inputConfirm.setError(getString(R.string.error_password_different));
            inputConfirm.requestFocus();
            inputConfirm.setText("");
            return false;
        }

        if (phone.isEmpty() || phone.length() < 9) {
            inputPhone.setError(getString(R.string.error_not_valid_phone));
            inputPhone.requestFocus();
            return false;
        }

        if (spUserType.getSelectedItemPosition() == 0) {
            Toast.makeText(requireContext(), getString(R.string.error_not_type_user), Toast.LENGTH_SHORT).show();
            spUserType.requestFocus();
            return false;
        }

        String userType = spUserType.getSelectedItem().toString();

        // 5. TODO LISTO PARA LA BASE DE DATOS
        // Si la ejecución llega a esta línea, los datos son 100% íntegros y válidos.

        // Ejemplo de inserción:
        // User newUser = new User(username, password, phone, userType);
        // long id = miDatabaseHelper.registrarUsuario(newUser);
        //
        // if (id > 0) {
        //     Toast.makeText(requireContext(), "Registro exitoso", Toast.LENGTH_SHORT).show();
        //     // Limpiar campos o navegar al Login
        // }
        SharedPreferences prefs = requireActivity().getSharedPreferences("QapacPrefs", Context.MODE_PRIVATE);
        SharedPreferences.Editor editor = prefs.edit();
        editor.putBoolean("first_time", false);
        editor.apply();
        return true;
    }
}