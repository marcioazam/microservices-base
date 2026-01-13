<?php

declare(strict_types=1);

namespace EmailService\Domain\ValueObject;

use EmailService\Domain\Exception\InvalidEmailException;

final readonly class Recipient
{
    public string $email;
    public ?string $name;

    public function __construct(string $email, ?string $name = null)
    {
        $email = trim(strtolower($email));
        
        if (!self::isValidFormat($email)) {
            throw new InvalidEmailException($email);
        }
        
        $this->email = $email;
        $this->name = $name !== null ? trim($name) : null;
    }

    public static function isValidFormat(string $email): bool
    {
        if (empty($email) || strlen($email) > 254) {
            return false;
        }

        // RFC 5322 compliant validation
        $pattern = '/^(?!(?:(?:\x22?\x5C[\x00-\x7E]\x22?)|(?:\x22?[^\x5C\x22]\x22?)){255,})' .
            '(?!(?:(?:\x22?\x5C[\x00-\x7E]\x22?)|(?:\x22?[^\x5C\x22]\x22?)){65,}@)' .
            '(?:(?:[\x21\x23-\x27\x2A\x2B\x2D\x2F-\x39\x3D\x3F\x5E-\x7E]+)|' .
            '(?:\x22(?:[\x01-\x08\x0B\x0C\x0E-\x1F\x21\x23-\x5B\x5D-\x7F]|' .
            '(?:\x5C[\x00-\x7F]))*\x22))(?:\.(?:(?:[\x21\x23-\x27\x2A\x2B\x2D\x2F-\x39\x3D\x3F\x5E-\x7E]+)|' .
            '(?:\x22(?:[\x01-\x08\x0B\x0C\x0E-\x1F\x21\x23-\x5B\x5D-\x7F]|' .
            '(?:\x5C[\x00-\x7F]))*\x22)))*@(?:(?:(?!.*[^.]{64,})' .
            '(?:(?:(?:xn--)?[a-z0-9]+(?:-[a-z0-9]+)*\.){1,126})' .
            '{1,}(?:(?:[a-z][a-z0-9]*)|(?:(?:xn--)[a-z0-9]+))(?:-[a-z0-9]+)*)|' .
            '(?:\[(?:(?:IPv6:(?:(?:[a-f0-9]{1,4}(?::[a-f0-9]{1,4}){7})|' .
            '(?:(?!(?:.*[a-f0-9][:\]]){7,})(?:[a-f0-9]{1,4}(?::[a-f0-9]{1,4}){0,5})?::' .
            '(?:[a-f0-9]{1,4}(?::[a-f0-9]{1,4}){0,5})?)))|' .
            '(?:(?:IPv6:(?:(?:[a-f0-9]{1,4}(?::[a-f0-9]{1,4}){5}:)|' .
            '(?:(?!(?:.*[a-f0-9]:){5,})(?:[a-f0-9]{1,4}(?::[a-f0-9]{1,4}){0,3})?::' .
            '(?:[a-f0-9]{1,4}(?::[a-f0-9]{1,4}){0,3}:)?)))?(?:(?:25[0-5])|' .
            '(?:2[0-4][0-9])|(?:1[0-9]{2})|(?:[1-9]?[0-9]))(?:\.(?:(?:25[0-5])|' .
            '(?:2[0-4][0-9])|(?:1[0-9]{2})|(?:[1-9]?[0-9]))){3}))\]))$/iD';

        // Use simpler validation for practical purposes
        return filter_var($email, FILTER_VALIDATE_EMAIL) !== false 
            && preg_match('/^[^@]+@[^@]+\.[^@]+$/', $email) === 1;
    }

    public function getDomain(): string
    {
        return substr($this->email, strpos($this->email, '@') + 1);
    }

    public function getLocalPart(): string
    {
        return substr($this->email, 0, strpos($this->email, '@'));
    }

    public function getFormatted(): string
    {
        if ($this->name !== null && $this->name !== '') {
            return sprintf('"%s" <%s>', $this->name, $this->email);
        }
        return $this->email;
    }

    public function equals(self $other): bool
    {
        return $this->email === $other->email;
    }

    public function __toString(): string
    {
        return $this->email;
    }
}
