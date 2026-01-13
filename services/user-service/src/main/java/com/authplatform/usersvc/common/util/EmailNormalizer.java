package com.authplatform.usersvc.common.util;

import org.springframework.stereotype.Component;
import java.util.Locale;

@Component
public class EmailNormalizer {

    public String normalize(String email) {
        if (email == null) {
            return null;
        }
        return email.trim().toLowerCase(Locale.ROOT);
    }

    public boolean isNormalized(String email) {
        if (email == null) {
            return true;
        }
        return email.equals(normalize(email));
    }
}
