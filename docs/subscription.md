# Subscription API

This API is used to view subscription plans and change a store's plan.

All endpoints require: `Authorization: Bearer <token>`

## GET /api/v1/subscriptions/plans

Returns all active subscription plans.

Example:

```bash
curl http://localhost:8080/api/v1/subscriptions/plans \
  -H "Authorization: Bearer <token>"
```

## GET /api/v1/stores/:storeID/subscription

Returns the current subscription for the store.

## PUT /api/v1/stores/:storeID/subscription

Changes the store's plan.

Request body:

```json
{
  "plan_code": "growth"
}
```

Example:

```bash
curl -X PUT http://localhost:8080/api/v1/stores/ad65bd37a3b3da7828ec9111/subscription \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InBlZWwyYXB1dEBnbWFpbC5jb20iLCJleHAiOjE3NzQyNTI4MTEsIm5hbWUiOiJwaGlyYXBoYXQga2xpbnRhbiIsInJvbGUiOiJvd25lciIsInN1YiI6ImFkNjViZDM3YTNiM2RhNzgyOGVjOTExMSJ9.srukC9ttNmpMjlqzVUpTc_RHBp6A2zHGRDD1NuIhUwE" \
  -H "Content-Type: application/json" \
  -d '{"plan_code":"growth"}'
```

## Notes

- Changing a plan closes the current active subscription first, then creates a new row
- Permission to change a plan requires role `owner`, `manager`, or `platform_admin`
