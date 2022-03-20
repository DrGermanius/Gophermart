### Abstract scheme of interaction with the system

Below is an abstract business logic of user interaction with the system:

1. User registers in "Gophermart" loyalty system.
2. User makes a purchase in "Gophermart" online store.
3. The order goes into "Gophermart" loyalty points system.
4. User transmits the number of the made order to the loyalty points system.
5. The system connects the order number with the user and checks the number with the loyalty points system. 
6. In case of positive loyalty points calculation, loyalty points are credited to the user's account.
7. User can make orders in "Gophermart" using loyalty points.

### Loyalty points calculation system

The loyalty points calculation system is an external service in the trusted loop. It operates according to the black box principle and is not available for inspection by external clients. The system calculates loyalty points for the committed order according to complicated algorithms, which can change at any moment.

An external customer can only see information about the number of loyalty points assigned for a certain order. An external consumer does not know the reasons for the presence or absence of the points.

### Endpoints of HTTP API

The "Gophermart" accumulative loyalty system must provide the following HTTP handlers:

* `POST /api/user/register` - user registration;
* `POST /api/user/login` - user authentication;
* `POST /api/user/orders` - loading the user's order number for calculation;
* `GET /api/user/orders` - receiving a list of order numbers uploaded by the user, their processing statuses and information about accruals;
* `GET /api/user/balance` - getting current balance of user's loyalty points account;
* `POST /api/user/balance/withdraw` - request to write off points from the accumulation account to pay for a new order;
* `GET /api/user/balance/withdrawals` - receiving information about the withdrawal of funds from the savings account by the user.

### General restrictions and requirements

* The data warehouse is PostgreSQL;
* The client can support HTTP requests/responses with data compression;
* Order numbers are unique and never repeated;
* An order number can only be accepted for processing once from one user;
* The order number can have no accrual;
* Remuneration is credited and spent in virtual points at the rate of 1 point = 1 dollar.

#### **User registration**

Handler: `POST /api/user/register`.

Registration is done with a login/password pair. Each login must be unique.
After successful registration there should be automatic user authentication.

The format of the request:

```
POST /api/user/register HTTP/1.1
Content-Type: application/json
...

{
	"login": "<login>",
	"password": "<password>"
}
```

Possible response codes:

- `200` - user successfully registered and authenticated;
- `400` - incorrect format of the request;
- `409` - login already taken;
- `500` - internal server error.

#### **User Authentication**

Handler: `POST /api/user/login`.

Authentication is performed by the login/password pair.

Request format:

```
POST /api/user/login HTTP/1.1
Content-Type: application/json
...

{
	"login": "<login>",
	"password": "<password>"
}
```

Possible response codes:

- `200` - user successfully authenticated;
- `400` - incorrect format of request;
- `401` - invalid login/password pair;
- `500` - internal server error.

#### **Load order number**

Handler: `POST /api/user/orders`.

Handler is available only to authenticated users. The order number is a sequence of digits of any length.

The order number can be checked for correctness using [Moon algorithm](https://en.wikipedia.org/wiki/Luhn_algorithm).

Query format:

```
POST /api/user/orders HTTP/1.1
Content-Type: text/plain
...

12345678903
```

Possible response codes:

- `200` - the order number has already been uploaded by this user;
- `202` - new order number accepted for processing;
- `400` - wrong format of the request;
- `401` - user is not authenticated;
- `409` - the order number has already been uploaded by another user;
- `422` - invalid format of the order number;
- `500` - internal server error.

#### **Get list of downloaded order numbers**

Handler: `GET /api/user/orders`.

The handler is available only to the authorized user. The order numbers in the output should be sorted by download time from the oldest to the newest. The date format is RFC3339.

Available billing processing statuses:

- `NEW` - the order has been uploaded to the system but has not gone into processing;
- `PROCESSING` - remuneration for the order is calculated;
- `INVALID` - the system for the calculation of remuneration refused in the calculation;
- `PROCESSED` - data on the order are checked and information on the calculation is successfully received.

Request format:

```
GET /api/user/orders HTTP/1.1
Content-Length: 0
```

Possible response codes:

- `200` - successful processing of the request.

  Response format:

    ```
    200 OK HTTP/1.1
    Content-Type: application/json
    ...
    
    [
    	{
            "number": "9278923470",
            "status": "PROCESSED",
            "accrual": 500,
            "uploaded_at": "2020-12-10T15:15:45+03:00"
        },
        {
            "number": "12345678903",
            "status": "PROCESSING",
            "uploaded_at": "2020-12-10T15:12:01+03:00"
        },
        {
            "number": "346436439",
            "status": "INVALID",
            "uploaded_at": "2020-12-09T16:09:53+03:00"
        }
    ]
    ```

- `204` - no data to answer.
- `401` - user is not authorized.
- `500` - internal server error.

#### **Get current user balance**

Handler: `GET /api/user/balance`.

Handler is available only for authorized user. The response should contain data about the current amount of loyalty points, as well as the amount of points used during the whole registration period.

The format of the request:

```
GET /api/user/balance HTTP/1.1
Content-Length: 0
```

Possible response codes:

- `200` - successful processing of the request.

  Response format:

    ```
    200 OK HTTP/1.1
    Content-Type: application/json
    ...
    
    {
    	"current": 500.5,
    	"withdrawn": 42
    }
    ```

- `401` - user is not authorized.
- `500` - internal server error.

#### **Request to debit funds**

Handler: `POST /api/user/balance/withdraw`.

The handler is only available to the authorized user. The order number is a hypothetical number of the user's new order in payment of which points are deducted.

Note: successful registration of the request is sufficient for successful deduction, no external accrual systems are provided and do not need to be implemented.

Request format:

```
POST /api/user/balance/withdraw HTTP/1.1
Content-Type: application/json

{
    "order": "2377225624",
    "sum": 751
}
```

Here `order` is the order number and `sum` is the amount of points to be charged from user's payment account.

Possible response codes:

- `200` - successful processing of the request;
- `401` - user is not authorized;
- `402` - insufficient funds on the account;
- `422` - invalid order number;
- `500` - internal server error.

#### **Receipt of withdrawal information**

Handler: `GET /api/user/balance/withdrawals`.

Handler is available only to the authorized user. The withdrawal facts in the output should be sorted by withdrawal time from the oldest to the newest. The date format is RFC3339.

Request format:

```
GET /api/user/withdrawals HTTP/1.1
Content-Length: 0
```

Possible response codes:

- `200` - successful processing of the request.

  Response format:

    ```
    200 OK HTTP/1.1
    Content-Type: application/json
    ...
    
    [
        {
            "order": "2377225624",
            "sum": 500,
            "processed_at": "2020-12-09T16:09:57+03:00"
        }
    ]
    ```

- `204` - no debit.
- `401` - user is not authorized.
- `500` - internal server error.

### Interaction with the system for calculating loyalty points

One handler is available for interaction with the system:

- `GET /api/orders/{number}` - getting information about the calculation of loyalty points.

Request format:

```
GET /api/orders/{number} HTTP/1.1
Content-Length: 0
```

Possible response codes:

- `200` - successful processing of the request.

  Response format:

    ```
    200 OK HTTP/1.1
    Content-Type: application/json
    ...
    
    {
        "order": "<number>",
        "status": "PROCESSED",
        "accrual": 500
    }
    ```

  The fields of the response object:

    - `order` - order number;
    - `status` - status of accrual calculation:

        - `REGISTERED` - the order is registered, but no accrual is not calculated;
        - `INVALID` - the order is not accepted for calculation and reward will not be calculated;
        - `PROCESSING` - charging is in process;
        - `PROCESSED` - computation of accrual is finished;

    - `accrual` - calculated points to accrual, if there is no accrual - there is no field in the answer.

- `429` - the number of requests to the service is exceeded.

  Answer format:

    ```
    429 Too Many Requests HTTP/1.1
    Content-Type: text/plain
    Retry-After: 60
    
    No more than N requests per minute allowed
    ```

- `500` is an internal server error.

The order can be taken into account at any moment after it has been made. The time of settlement is not regulated by the system. Statuses `INVALID` and `PROCESSED` are final.

The total number of requests for information on the charge is not limited.

### Configuration of the accumulative loyalty system service

The service must support the following configuration methods:

- service launch address and port: OS environment variable `RUN_ADDRESS` or flag `a`
- database connection address: OS environment variable `DATABASE_URI` or flag `-d`
- charging system address: OS environment variable `ACCRUAL_SYSTEM_ADDRESS` or flag `-r`
