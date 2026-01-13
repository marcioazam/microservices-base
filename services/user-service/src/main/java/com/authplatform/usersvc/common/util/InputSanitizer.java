package com.authplatform.usersvc.common.util;

import org.springframework.stereotype.Component;
import java.util.regex.Pattern;

@Component
public class InputSanitizer {

    private static final Pattern SCRIPT_PATTERN = Pattern.compile(
            "<script[^>]*>.*?</script>", Pattern.CASE_INSENSITIVE | Pattern.DOTALL
    );
    private static final Pattern HTML_TAG_PATTERN = Pattern.compile("<[^>]+>");
    private static final Pattern SQL_INJECTION_PATTERN = Pattern.compile(
            "('|--|;|/\\*|\\*/|xp_|sp_|0x)", Pattern.CASE_INSENSITIVE
    );

    public String sanitize(String input) {
        if (input == null) {
            return null;
        }

        String sanitized = input;
        sanitized = SCRIPT_PATTERN.matcher(sanitized).replaceAll("");
        sanitized = HTML_TAG_PATTERN.matcher(sanitized).replaceAll("");
        sanitized = sanitized.replace("&", "&amp;")
                .replace("<", "&lt;")
                .replace(">", "&gt;")
                .replace("\"", "&quot;")
                .replace("'", "&#x27;");

        return sanitized;
    }

    public boolean containsPotentialInjection(String input) {
        if (input == null) {
            return false;
        }
        return SQL_INJECTION_PATTERN.matcher(input).find() ||
                SCRIPT_PATTERN.matcher(input).find();
    }

    public String sanitizeForDisplay(String input) {
        if (input == null) {
            return null;
        }
        return input.replace("<", "&lt;")
                .replace(">", "&gt;")
                .replace("\"", "&quot;")
                .replace("'", "&#x27;");
    }
}
