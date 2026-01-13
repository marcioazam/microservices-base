<?php

declare(strict_types=1);

namespace EmailService\Application\Util;

/**
 * Centralized PII masking utility.
 * Single source of truth for all PII masking operations.
 */
final readonly class PiiMasker
{
    private const MASK_CHAR = '*';
    private const MASK_LENGTH = 3;

    /**
     * Mask an email address, preserving first character and domain.
     * 
     * Examples:
     * - john.doe@example.com -> j***@example.com
     * - a@example.com -> *@example.com
     * - ab@example.com -> a***@example.com
     */
    public static function maskEmail(string $email): string
    {
        $parts = explode('@', $email);

        if (count($parts) !== 2) {
            return str_repeat(self::MASK_CHAR, self::MASK_LENGTH);
        }

        $local = $parts[0];
        $domain = $parts[1];

        $maskedLocal = match (true) {
            strlen($local) === 0 => self::MASK_CHAR,
            strlen($local) === 1 => self::MASK_CHAR,
            default => $local[0] . str_repeat(self::MASK_CHAR, self::MASK_LENGTH),
        };

        return $maskedLocal . '@' . $domain;
    }

    /**
     * Mask a phone number, preserving last 4 digits.
     * 
     * Examples:
     * - +1234567890 -> ******7890
     * - 1234 -> 1234 (too short to mask)
     */
    public static function maskPhone(string $phone): string
    {
        $digits = preg_replace('/\D/', '', $phone);

        if ($digits === null || strlen($digits) <= 4) {
            return $phone;
        }

        $visibleDigits = 4;
        $maskedLength = strlen($digits) - $visibleDigits;
        $lastDigits = substr($digits, -$visibleDigits);

        return str_repeat(self::MASK_CHAR, $maskedLength) . $lastDigits;
    }

    /**
     * Mask a name, preserving first character of each word.
     * 
     * Examples:
     * - John Doe -> J*** D**
     * - Alice -> A****
     */
    public static function maskName(string $name): string
    {
        if (strlen($name) === 0) {
            return '';
        }

        $words = explode(' ', $name);
        $maskedWords = array_map(
            fn(string $word) => self::maskWord($word),
            $words
        );

        return implode(' ', $maskedWords);
    }

    /**
     * Mask an IP address.
     * 
     * Examples:
     * - 192.168.1.100 -> 192.168.*.*
     * - 2001:0db8:85a3::8a2e:0370:7334 -> 2001:0db8:****:****
     */
    public static function maskIpAddress(string $ip): string
    {
        // IPv4
        if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV4)) {
            $parts = explode('.', $ip);
            if (count($parts) === 4) {
                return $parts[0] . '.' . $parts[1] . '.' . self::MASK_CHAR . '.' . self::MASK_CHAR;
            }
        }

        // IPv6
        if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV6)) {
            $parts = explode(':', $ip);
            if (count($parts) >= 4) {
                return $parts[0] . ':' . $parts[1] . ':' . str_repeat(self::MASK_CHAR, 4) . ':' . str_repeat(self::MASK_CHAR, 4);
            }
        }

        return str_repeat(self::MASK_CHAR, 8);
    }

    /**
     * Mask a credit card number, preserving last 4 digits.
     * 
     * Examples:
     * - 4111111111111111 -> ************1111
     */
    public static function maskCreditCard(string $cardNumber): string
    {
        $digits = preg_replace('/\D/', '', $cardNumber);

        if ($digits === null || strlen($digits) < 4) {
            return str_repeat(self::MASK_CHAR, 16);
        }

        $lastFour = substr($digits, -4);
        $maskedLength = strlen($digits) - 4;

        return str_repeat(self::MASK_CHAR, $maskedLength) . $lastFour;
    }

    /**
     * Mask arbitrary sensitive data.
     * Shows first and last character with masks in between.
     */
    public static function maskSensitive(string $data, int $visibleChars = 2): string
    {
        $length = strlen($data);

        if ($length <= $visibleChars * 2) {
            return str_repeat(self::MASK_CHAR, $length);
        }

        $start = substr($data, 0, $visibleChars);
        $end = substr($data, -$visibleChars);
        $maskLength = $length - ($visibleChars * 2);

        return $start . str_repeat(self::MASK_CHAR, $maskLength) . $end;
    }

    private static function maskWord(string $word): string
    {
        $length = strlen($word);

        return match (true) {
            $length === 0 => '',
            $length === 1 => $word,
            default => $word[0] . str_repeat(self::MASK_CHAR, $length - 1),
        };
    }
}
