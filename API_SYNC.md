# API Synchronization Status

This document tracks the synchronization between frontend (`inventory-mgmt-fe`) and backend (`inventory-mgmt-be`).

## Model Alignment

### User Model
✅ **Synchronized**
- Backend: `UserModel` → JSON: `User`
- All fields match frontend `User` interface
- `updatedAt` is omitted from JSON responses (`json:"-"`)
- `createdAt` serializes as ISO 8601 string (RFC3339)

### Product Model
✅ **Synchronized**
- Backend: `Product` → JSON: `Product`
- All fields match frontend `Product` interface
- `updatedAt` is omitted from JSON responses (`json:"-"`)
- `createdAt` serializes as ISO 8601 string (RFC3339)
- `attributes` field supports dynamic key-value pairs

### Sale Model
✅ **Synchronized**
- Backend: `Sale` → JSON: `Sale`
- All fields match frontend `Sale` interface
- `updatedAt` is omitted from JSON responses (`json:"-"`)
- `createdAt` serializes as ISO 8601 string (RFC3339)
- `productAttributes` field supports dynamic key-value pairs

## API Endpoints

### Authentication
| Method | Endpoint | Frontend | Backend | Status |
|--------|----------|----------|---------|--------|
| POST | `/api/v1/users/login` | ✅ | ✅ | ✅ Synced |

### Users
| Method | Endpoint | Frontend | Backend | Status |
|--------|----------|----------|---------|--------|
| GET | `/api/v1/users` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/users/:id` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/users/branch/:branch` | ✅ | ✅ | ✅ Synced |
| POST | `/api/v1/users` | ✅ | ✅ | ✅ Synced |
| PUT | `/api/v1/users/:id` | ✅ | ✅ | ✅ Synced |
| DELETE | `/api/v1/users/:id` | ✅ | ✅ | ✅ Synced |

### Products
| Method | Endpoint | Frontend | Backend | Status |
|--------|----------|----------|---------|--------|
| GET | `/api/v1/products` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/products/:id` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/products/branch/:branch` | ✅ | ✅ | ✅ Synced |
| POST | `/api/v1/products` | ✅ | ✅ | ✅ Synced |
| PUT | `/api/v1/products/:id` | ✅ | ✅ | ✅ Synced |
| DELETE | `/api/v1/products/:id` | ✅ | ✅ | ✅ Synced |
| POST | `/api/v1/products/:id/reduce?quantity=X` | ✅ | ✅ | ✅ Synced |

### Sales
| Method | Endpoint | Frontend | Backend | Status |
|--------|----------|----------|---------|--------|
| GET | `/api/v1/sales` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/sales/:id` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/sales/user/:userId` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/sales/branch/:branch` | ✅ | ✅ | ✅ Synced |
| GET | `/api/v1/sales/date-range?startDate=X&endDate=Y` | ✅ | ✅ | ✅ Synced |
| POST | `/api/v1/sales` | ✅ | ✅ | ✅ Synced |
| PUT | `/api/v1/sales/:id` | ✅ | ✅ | ✅ Synced |
| DELETE | `/api/v1/sales/:id` | ✅ | ✅ | ✅ Synced |

## Request/Response Formats

### Login Request
```json
{
  "username": "string",
  "password": "string"
}
```

### Login Response
```json
{
  "user": {
    "id": "string",
    "name": "string",
    "username": "string",
    "password": "string",
    "role": "super_admin" | "user",
    "branch": "string",
    "email": "string?",
    "phone": "string?",
    "profilePictureUri": "string?",
    "syncStatus": "online" | "offline" | "synced"?,
    "createdAt": "ISO 8601 string"
  },
  "token": "JWT token string"
}
```

### User Response
```json
{
  "id": "string",
  "name": "string",
  "username": "string",
  "password": "string",
  "role": "super_admin" | "user",
  "branch": "string",
  "email": "string?",
  "phone": "string?",
  "profilePictureUri": "string?",
  "syncStatus": "online" | "offline" | "synced"?,
  "createdAt": "ISO 8601 string"
}
```

### Product Response
```json
{
  "id": "string",
  "name": "string",
  "price": "number",
  "currency": "string",
  "branch": "string",
  "quantity": "number",
  "imageUri": "string?",
  "syncStatus": "online" | "offline" | "synced"?,
  "attributes": {
    "key": "any value"
  },
  "createdAt": "ISO 8601 string"
}
```

### Sale Response
```json
{
  "id": "string",
  "productId": "string",
  "productName": "string",
  "productAttributes": {
    "key": "any value"
  },
  "quantity": "number",
  "unitPrice": "number",
  "totalPrice": "number",
  "currency": "string",
  "sellerId": "string",
  "sellerName": "string",
  "branch": "string",
  "paymentStatus": "credit" | "promised",
  "syncStatus": "online" | "offline" | "synced"?,
  "createdAt": "ISO 8601 string"
}
```

## Authentication

- ✅ JWT tokens are generated on login
- ✅ Tokens are validated via `AuthMiddleware()`
- ✅ Protected routes require `Authorization: Bearer <token>` header
- ✅ Token expiration is configurable via `JWT_EXPIRATION_HOURS`

## Data Types

### Enums
- `UserRole`: `"super_admin" | "user"` ✅ Synced
- `SyncStatus`: `"online" | "offline" | "synced"` ✅ Synced
- `PaymentStatus`: `"credit" | "promised"` ✅ Synced

### Date Format
- All `createdAt` fields: ISO 8601 / RFC3339 format ✅
- Example: `"2024-01-15T10:30:00Z"`

## Notes

1. **UpdatedAt Field**: The `updatedAt` field exists in backend models but is omitted from JSON responses using `json:"-"` tag to match frontend expectations.

2. **Dynamic Attributes**: Both `Product.attributes` and `Sale.productAttributes` support dynamic key-value pairs as `Record<string, any>` in TypeScript and `map[string]interface{}` in Go.

3. **Response Wrapping**: List endpoints return arrays wrapped in objects:
   - `GET /users` → `{ "users": [...] }`
   - `GET /products` → `{ "products": [...] }`
   - `GET /sales` → `{ "sales": [...] }`

4. **Error Format**: All errors return:
   ```json
   {
     "error": "error message string"
   }
   ```

## Last Synced
- Date: 2024-01-15
- Status: ✅ Fully Synchronized
