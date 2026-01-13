package com.authplatform.usersvc.infra.persistence;

import com.authplatform.usersvc.domain.model.EmailVerificationToken;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface EmailVerificationTokenRepository extends JpaRepository<EmailVerificationToken, UUID> {

    Optional<EmailVerificationToken> findByTokenHash(String tokenHash);

    List<EmailVerificationToken> findByUserIdAndUsedAtIsNull(UUID userId);

    @Modifying
    @Query("UPDATE EmailVerificationToken t SET t.usedAt = CURRENT_TIMESTAMP WHERE t.userId = :userId AND t.usedAt IS NULL")
    int invalidateUnusedTokensForUser(@Param("userId") UUID userId);

    @Query("SELECT COUNT(t) FROM EmailVerificationToken t WHERE t.userId = :userId AND t.createdAt > :since")
    long countRecentTokensForUser(@Param("userId") UUID userId, @Param("since") java.time.Instant since);
}
