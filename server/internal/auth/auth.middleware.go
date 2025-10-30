package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tylerjvollick/nori/internal/database"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

// AuthMiddleware verifies JWT and attaches claims to context.
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}

		// Repositories
		userRepo := repositories.NewUserRepository(database.DB)
		accountRepo := repositories.NewAccountRepository(database.DB)
		userAccountRepo := repositories.NewUserAccountRepository(database.DB)
		// Services
		authService := services.NewAuthService(userRepo, accountRepo, userAccountRepo)

		authDTO, err := authService.Authenticate(authHeader)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized...?",
			})
		}

		// return authDTO, nil
		c.Locals("authDTO", authDTO)

		return c.Next()
	}
}

// RequirePermissions ensures the user has required access for a resource.
func RequirePermissions(resource PermissionType, required Access) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userClaims, ok := c.Locals("user").(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found in context",
			})
		}

		roleVal, ok := userClaims["role"].(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Missing or invalid role in token",
			})
		}

		roleName := Role(roleVal)
		rolePerms, ok := RoleDefinitions[roleName]
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Role not defined",
			})
		}

		userAccess, ok := rolePerms[resource]
		if !ok || userAccess&required != required {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		return c.Next()
	}
}
