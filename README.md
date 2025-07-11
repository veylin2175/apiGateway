# TrustVote: Децентрализованная Платформа для Голосования

## 1\. Название проекта и описание

**TrustVote** — это децентрализованная платформа для голосования, построенная на блокчейне Ethereum. Она использует смарт-контракты для безопасного и прозрачного голосования, а также микросервисную архитектуру для масштабируемой обработки данных на бэкенде. Это DApp призвано обеспечить надежную систему для создания, участия и управления голосованиями с интегрированными механизмами стейкинга для стимулирования и обеспечения безопасности.

Система состоит из:

  * **Смарт-контрактов (Solidity):** Основная логика для голосования, стейкинга и распределения наград.
  * **Go API Gateway:** Служит основным интерфейсом для фронтенда, обрабатывая веб-запросы и управляя взаимодействием с блокчейном и бэкенд-сервисами.
  * **Java Kafka Service:** Надежный бэкенд-сервис, который потребляет и производит сообщения Kafka, взаимодействует с базой данных и, возможно, выполняет длительную или сложную бизнес-логику, связанную с данными голосования, историей пользователей и, возможно, прослушиванием событий блокчейна.
  * **Фронтенд (HTML, CSS, JavaScript):** Удобный веб-интерфейс для взаимодействия с DApp.
  * **Kafka:** Распределенная потоковая платформа, используемая для асинхронной связи между Go API Gateway и Java Kafka Service.
  * **PostgreSQL:** Реляционная база данных, используемая Java Kafka Service для хранения данных голосования, профилей пользователей и истории.

-----

## 2\. Возможности

  * **Децентрализованное голосование:** Основная логика голосования реализована на смарт-контракте для прозрачности и неизменяемости.
  * **Механизм стейкинга:** Пользователи могут стейкать ETH для участия в определенных действиях (например, создание голосований с высокими ставками, получение наград).
  * **Получение наград:** Стейкеры могут получать накопленные токены-награды через DApp.
  * **Профили пользователей:** Просмотр личной информации о стейкинге, созданных голосованиях и истории голосования.
  * **API Gateway:** Единая точка входа для фронтенд-коммуникации, абстрагирующая сложную бэкенд-логику.
  * **Асинхронная обработка:** Kafka используется для эффективной, неблокирующей связи между сервисами, улучшая масштабируемость и оперативность.
  * **Модульная архитектура:** Разделение ответственности (взаимодействие с блокчейном, хранение данных, фронтенд) для упрощения разработки и поддержки.

-----

## 3\. Предварительные требования

Прежде чем начать, убедитесь, что у вас установлены следующие компоненты:

  * **Git:** Для клонирования репозитория.
  * **Docker и Docker Compose:** Для запуска Kafka и PostgreSQL (рекомендуется).
  * **Go (1.22+):** Для API Gateway.
  * **Java Development Kit (JDK 17+):** Для Kafka Service.
  * **Maven (3.8+):** Для сборки Java Kafka Service.
  * **Node.js (18+):** Для фронтенд-зависимостей (если используется инструмент сборки, такой как Webpack/Vite, хотя здесь используется чистый JS).
  * **npm или Yarn:** Менеджер пакетов для Node.js.
  * **Solidity Compiler (solc):** Для компиляции смарт-контрактов (часто входит в состав Hardhat/Foundry/Truffle).
  * **Hardhat / Foundry / Truffle (Рекомендуется):** Для локальной разработки блокчейна, тестирования и развертывания смарт-контрактов. В этом руководстве мы будем использовать Anvil (из Foundry).
  * **MetaMask (расширение для браузера):** Для взаимодействия с DApp из браузера.

-----

## 4\. Руководство по установке и настройке

Выполните следующие шаги, чтобы запустить TrustVote локально.

### 4.0. Инициализация проекта и зависимостей (однократно)

1.  **Клонируйте репозиторий:**

    ```bash
    git clone https://github.com/your-username/trustvote.git
    cd trustvote
    ```

2.  **Запустите Kafka и PostgreSQL с помощью Docker Compose:**
    Перейдите в корневой каталог вашего проекта, где находится файл `docker-compose.yml` (возможно, вам придется создать его, если он не предоставлен, или убедиться, что он находится в каталоге `deploy/` и т.д.).

    ```bash
    docker-compose up -d kafka zookeeper postgres pgadmin # Настройте имена сервисов в соответствии с вашим docker-compose.yml
    ```

      * Дождитесь запуска всех сервисов. Вы можете проверить их статус с помощью `docker-compose ps`.

3.  **Создайте Kafka-топики:**
    Вам нужно будет создать Kafka-топики, используемые вашими сервисами. Это можно сделать вручную с помощью инструментов Kafka или автоматически с помощью Spring Kafka, если он настроен. Предположим ручное создание для ясности:

    ```bash
    # Подключитесь к контейнеру Kafka (замените <kafka-container-id> на фактический ID из `docker-compose ps`)
    docker exec -it <kafka-container-id> bash

    # Создайте топики (настройте имена топиков в соответствии с вашей конфигурацией)
    kafka-topics --create --topic user_data_request --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic user_data_response --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic vote_cast_event --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic create_voting_request --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic get_voting_info_request --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic voting_info_response --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic get_all_votings_request --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    kafka-topics --create --topic all_votings_response --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
    # Выйдите из контейнера Kafka
    exit
    ```

### 4.1. Смарт-контракты

1.  **Запустите локальную цепочку блоков Ethereum (например, Anvil):**

    ```bash
    anvil --port 8545 # Или `npx ganache-cli` или `hardhat node`
    ```

      * Оставьте этот терминал открытым. Запишите приватные ключи сгенерированных аккаунтов. Один из них понадобится для вашего Go API Gateway.

2.  **Скомпилируйте смарт-контракты:**
    Перейдите в каталог вашего проекта со смарт-контрактами (например, `contracts/` или `blockchain/`).

    ```bash
    # Если используете Foundry
    forge build

    # Если используете Hardhat
    npx hardhat compile
    ```

3.  **Разверните смарт-контракты:**
    Разверните `StakeManager.sol`, `Voting.sol` и `RewardToken.sol` на вашем локальном экземпляре Anvil.

      * **Критически важно:** Запишите развернутые адреса `StakeManager` и `RewardToken`.
      * Вам нужно будет установить адрес `RewardToken` в конструкторе/инициализации `StakeManager`, если это не делается автоматически.
      * **Важно:** После развертывания `RewardToken` переведите некоторое количество токенов на адрес контракта `StakeManager`. Функция `getTokens` завершится ошибкой, если `StakeManager` не будет иметь `RewardToken` для распределения.
          * Пример (с использованием простого скрипта Hardhat/Foundry):
            ```solidity
            // В вашем скрипте развертывания
            RewardToken rewardToken = new RewardToken();
            StakeManager stakeManager = new StakeManager(address(rewardToken)); // Передаем адрес токена в StakeManager

            // Сначала выпустите токены для развернувшегося
            rewardToken.mint(msg.sender, 1000000 * 10**rewardToken.decimals());
            // Разрешите StakeManager тратить токены от развернувшегося (если StakeManager забирает награды)
            // rewardToken.approve(address(stakeManager), some_large_amount);
            // ИЛИ напрямую переведите токены на адрес контракта StakeManager
            rewardToken.transfer(address(stakeManager), 100000 * 10**rewardToken.decimals()); // Отправляем 100k токенов в StakeManager
            ```

4.  **Сгенерируйте Go-биндинги:**
    Перейдите в каталог `internal/contracts` вашего Go-проекта (или туда, где хранятся сгенерированные биндинги контрактов).

    ```bash
    # Предполагая, что у вас есть файлы abi/bin в ./build/contracts
    abigen --abi <путь_к_StakeManager.abi> --bin <путь_к_StakeManager.bin> --pkg voting --out voting/StakeManager.go
    abigen --abi <путь_к_Voting.abi> --bin <путь_к_Voting.bin> --pkg voting --out voting/Voting.go
    # Возможно, вам придется изменить имя пакета в зависимости от вашей структуры. Будем считать `voting`, как в предыдущих примерах.
    # Убедитесь, что RewardToken также связан, если вы напрямую взаимодействуете с ним из Go.
    ```

    Убедитесь, что имя пакета `voting` в Go-биндингах соответствует импортам вашего Go-кода (`apiGateway/internal/contracts/voting`).

### 4.2. Java Kafka Service

1.  **Перейдите в каталог Java-проекта:**

    ```bash
    cd <путь_к_вашему_java_сервису> # например, java-kafka-service/
    ```

2.  **Настройте `application.properties` / `application.yml`:**
    Обновите конфигурации базы данных, Kafka и любые другие специфичные для сервиса настройки.

      * **Критически важно:** Если вы столкнетесь с ошибками определения бинов, установите `spring.main.allow-bean-definition-overriding=true`.
      * **Кодировка:** Добавьте `server.servlet.encoding.charset=UTF-8` и убедитесь, что ваш фреймворк логирования (Logback/Log4j) настроен на UTF-8. Для вывода в консоль убедитесь, что ваш терминал также настроен на UTF-8.
      * **База данных:** Убедитесь, что `spring.datasource.url`, `username` и `password` соответствуют вашей настройке PostgreSQL в Docker.
      * **Kafka:** Убедитесь, что `spring.kafka.bootstrap-servers` указывает на ваш Kafka-брокер (например, `localhost:9092`).
      * **JPA/Hibernate:** Убедитесь, что `spring.jpa.hibernate.ddl-auto` установлен правильно (например, `update` для разработки, `none` для продакшена) и другие свойства JPA верны.

3.  **Соберите проект:**

    ```bash
    mvn clean install
    ```

4.  **Запустите сервис:**

    ```bash
    java -jar target/<имя_вашего_jar-файла_сервиса>.jar
    ```

      * Оставьте этот терминал открытым.

### 4.3. Go API Gateway

1.  **Перейдите в каталог Go-проекта:**

    ```bash
    cd <путь_к_вашему_go_шлюзу> # например, apiGateway/
    ```

2.  **Настройте `config/config.yaml`:**
    Обновите файл конфигурации с правильными адресами и настройками.

      * `Blockchain.RpcUrl`: Ваш RPC-URL Anvil (например, `http://localhost:8545`).
      * `Blockchain.PrivateKey`: Приватный ключ аккаунта Anvil, у которого есть ETH для газа. Это будет адрес, который Go API Gateway использует для отправки транзакций (стейкинг, анстейкинг, получение наград).
      * `Blockchain.StakeManagerContractAddress`: Адрес вашего развернутого контракта `StakeManager.sol`.
      * `Kafka.Broker`: Адрес вашего Kafka-брокера (например, `localhost:9092`).
      * `HTTPServer.Address`: Адрес, на котором API Gateway будет слушать (например, `localhost:8080`).
      * **Имена Kafka-топиков:** Убедитесь, что `Kafka.Topics` совпадают с именами топиков, которые вы создали ранее.

3.  **Загрузите Go-зависимости:**

    ```bash
    go mod tidy
    ```

4.  **Запустите API Gateway:**

    ```bash
    go run cmd/main/main.go
    ```

      * Оставьте этот терминал открытым.

### 4.4. Фронтенд

1.  **Обслуживание HTML-файлов:**
    Фронтенд состоит из статических HTML, CSS и JavaScript файлов. Вам нужен веб-сервер для их обслуживания.

      * **Вариант A (Python SimpleHTTPServer):**
        ```bash
        cd frontend/ # Или туда, где находятся ваши HTML/CSS/JS
        python3 -m http.server 8000
        ```
      * **Вариант B (Интеграция с Go Gateway):** Вы можете настроить свой Go API Gateway для обслуживания статических файлов. Это распространенный подход для развертывания в одном бинарном файле. (Не рассматривается в этом README, но является хорошим улучшением).
      * **Вариант C (Использование `live-server` для разработки):**
        ```bash
        npm install -g live-server
        cd frontend/
        live-server
        ```

2.  **Откройте в браузере:**
    После запуска сервера откройте браузер и перейдите по адресу `http://localhost:8000/profile.html` (или по URL, предоставленному `live-server`).

-----

## 5\. Конфигурация

Все критические конфигурации для Go API Gateway управляются в `apiGateway/config/config.yaml`.
Java Kafka Service использует `application.properties` или `application.yml` для своей конфигурации.

**Ключевые параметры конфигурации:**

**Go API Gateway (`config/config.yaml`):**

```yaml
env: "dev" # "dev" или "prod"

http_server:
  address: "localhost:8080" # Адрес, на котором API Gateway будет слушать
  timeout: 4s
  idle_timeout: 60s

blockchain:
  rpc_url: "http://localhost:8545" # Ваш RPC-URL Anvil/Ganache/Sepolia
  private_key: "ВАШ_ПРИВАТНЫЙ_КЛЮЧ_АККАУНТА_METAMASK_ИЗ_ANVIL" # Приватный ключ с ETH для газа
  stake_manager_contract_address: "0x..." # Развернутый адрес StakeManager.sol
  # Добавьте другие адреса контрактов при необходимости, например, RewardTokenContractAddress

kafka:
  broker: "localhost:9092" # Адрес Kafka-брокера
  group_id: "api_gateway_consumer_group"
  topics:
    user_data_request: "user_data_request"
    user_data_response: "user_data_response"
    vote_cast_event: "vote_cast_event"
    create_voting_request: "create_voting_request"
    get_voting_info_request: "get_voting_info_request"
    voting_info_response: "voting_info_response"
    get_all_votings_request: "get_all_votings_request"
    all_votings_response: "all_votings_response"
```

**Java Kafka Service (`src/main/resources/application.properties`):**

```properties
spring.kafka.bootstrap-servers=localhost:9092
spring.kafka.consumer.group-id=java_kafka_service_consumer_group
spring.kafka.producer.key-serializer=org.apache.kafka.common.serialization.StringSerializer
spring.kafka.producer.value-serializer=org.springframework.kafka.support.serializer.JsonSerializer
spring.kafka.consumer.key-deserializer=org.apache.kafka.common.serialization.StringDeserializer
spring.kafka.consumer.value-deserializer=org.springframework.kafka.support.serializer.JsonDeserializer
spring.kafka.properties.spring.json.trusted.packages=*

# Конфигурация базы данных (PostgreSQL)
spring.datasource.url=jdbc:postgresql://localhost:5432/your_database_name
spring.datasource.username=your_db_user
spring.datasource.password=your_db_password
spring.datasource.driver-class-name=org.postgresql.Driver

# Конфигурация Hibernate
spring.jpa.hibernate.ddl-auto=update # Используйте 'update' для разработки, 'none' для продакшена
spring.jpa.properties.hibernate.dialect=org.hibernate.dialect.PostgreSQLDialect
spring.jpa.show-sql=true
spring.jpa.properties.hibernate.format_sql=true

# Порт сервера (если работает как отдельный сервис с HTTP-эндпоинтами)
server.port=8081 # Пример, если он предоставляет свой собственный API для внутреннего использования

# Настройки кодировки для логов и вывода
server.servlet.encoding.charset=UTF-8
logging.charset.console=UTF-8
logging.charset.file=UTF-8

# Разрешить переопределение определения бинов (часто необходимо для тестирования или сложных настроек)
spring.main.allow-bean-definition-overriding=true

# Добавьте специфичные конфигурации Kafka-топиков при необходимости
kafka.topics.user-data-request=user_data_request
kafka.topics.user-data-response=user_data_response
# ... и так далее для всех топиков
```

-----

## 6\. Использование

После того как все сервисы запущены и фронтенд обслуживается, вы можете взаимодействовать с DApp.

### 6.1. Общий процесс

1.  **Откройте `profile.html`** в вашем браузере.
2.  **Нажмите "Подключить MetaMask":** Вам будет предложено подключить ваш кошелек MetaMask. Выберите аккаунт из вашего экземпляра Anvil.
3.  После успешного подключения появятся адрес вашего кошелька и начальные данные профиля.

### 6.2. Стейкинг ETH

1.  На странице профиля нажмите **"Stake ETH"**.
2.  Введите сумму ETH, которую вы хотите застейкать (например, `0.0001`).
3.  Подтвердите транзакцию в MetaMask.
4.  API Gateway отправит транзакцию в блокчейн.

### 6.3. Анстейкинг ETH

1.  На странице профиля нажмите **"Unstake ETH"**.
2.  Подтвердите запрос на вывод всех ваших застейканных ETH.
3.  Подтвердите транзакцию в MetaMask.
4.  API Gateway отправит транзакцию анстейкинга.

### 6.4. Получение наград

1.  На странице профиля нажмите **"Получить награды"**.
2.  Подтвердите запрос.
3.  Подтвердите транзакцию в MetaMask.
4.  API Gateway отправит транзакцию получения наград в контракт `StakeManager`.
      * **Примечание:** Это сработает только в том случае, если у вас накоплены награды и прошел период охлаждения, определенный в вашем контракте `StakeManager.sol`, а также если контракт содержит достаточно токенов награды для распределения.

### 6.5. Подключение кошелька и профиль

  * Будут отображены адрес вашего кошелька и статистика голосований.
  * Таблица "Ваша история голосований" будет заполнена данными с бэкенда.
  * Нажмите **"Выйти из аккаунта"**, чтобы отключить ваш кошелек MetaMask.

### 6.6. Создание голосований (будущее/администратор)

  * Эта функциональность должна быть реализована. В настоящее время существует Kafka-топик для `create_voting_request`. Вы должны создать форму пользовательского интерфейса, которая отправляет POST-запрос на ваш API Gateway, который затем публикует его в этом Kafka-топике. Java-сервис будет потреблять его и сохранять голосование в БД, потенциально взаимодействуя с контрактом `Voting.sol`.

### 6.7. Участие в голосованиях

  * Эта функциональность подразумевается Kafka-топиком `vote_cast_event`. Вы будете взаимодействовать с интерфейсом голосования (не полностью описанным в текущем HTML-примере), который отправляет запросы в API Gateway. API Gateway публикует `vote_cast_event` в Kafka. Java-сервис потребляет его, чтобы записать голос в БД, и потенциально запускает блокчейн-транзакцию для голосования в цепочке.

-----

## 7\. API-эндпоинты

Go API Gateway предоставляет следующие эндпоинты:

| Метод | Путь                           | Описание                                             | Тело запроса (JSON)                                       | Тело ответа (JSON)                                                  |
| :---- | :----------------------------- | :--------------------------------------------------- | :-------------------------------------------------------- | :-------------------------------------------------------------------- |
| `POST` | `/staking`                     | Стейкает ETH по указанному адресу.                   | `{ "amount": <float>, "staker_address": "0x..." }`        | `{ "status": 200, "message": "...", "data": { "tx_hash": "0x..." } }` |
| `POST` | `/unstake`                     | Анстейкает весь ETH по указанному адресу.            | `{ "staker_address": "0x..." }`                            | `{ "status": 200, "message": "...", "data": { "tx_hash": "0x..." } }` |
| `POST` | `/profile/get_tokens`          | Получает накопленные токены-награды для адреса, настроенного в API Gateway. | `{}` (Пустое, адрес берется из конфигурации бэкенда)            | `{ "status": 200, "message": "...", "data": { "tx_hash": "0x..." } }` |
| `POST` | `/user-data`                   | Получает данные стейкинга и историю голосования для пользователя. | `{ "user_address": "0x..." }`                              | `{ "status": 200, "message": "...", "data": { ...user_data... } }` |
| `POST` | `/connect-wallet`              | Уведомляет бэкенд о подключении кошелька (для логирования/отслеживания). | `{ "walletAddress": "0x..." }`                           | `text/plain` или базовый JSON-статус                                |
| `POST` | `/vote`                        | Отправляет голос за определенный вариант в голосовании. | `{ "voting_id": "123", "option_id": "1", "voter_address": "0x..." }` | `{ "status": 200, "message": "..." }`                                |
| `POST` | `/create-voting`               | Создает новое голосование.                             | `{ "title": "...", "description": "...", "options": [...] }` | `{ "status": 200, "message": "..." }`                                |
| `GET`  | `/votings/{id}`                | Получает подробную информацию о конкретном голосовании. | (Параметр пути `id`)                                     | `{ "status": 200, "message": "...", "data": { ...voting_details... } }` |
| `GET`  | `/votings/all`                 | Получает список последних голосований.                 | (Нет)                                                    | `{ "status": 200, "message": "...", "data": { "votings": [...] } }` |

-----

## 8\. Kafka-топики

Для межсервисной связи используются следующие Kafka-топики:

| Имя топика                  | Производитель                 | Потребитель                 | Описание                                                                  |
| :-------------------------- | :---------------------------- | :-------------------------- | :------------------------------------------------------------------------ |
| `user_data_request`         | Go API Gateway                | Java Kafka Service          | Запрос на данные профиля пользователя и историю.                          |
| `user_data_response`        | Java Kafka Service            | Go API Gateway              | Ответ, содержащий данные профиля пользователя и историю.                  |
| `vote_cast_event`           | Go API Gateway                | Java Kafka Service          | Событие, указывающее на то, что пользователь проголосовал.                |
| `create_voting_request`     | Go API Gateway                | Java Kafka Service          | Запрос на создание нового голосования.                                    |
| `get_voting_info_request`   | Go API Gateway                | Java Kafka Service          | Запрос на подробную информацию о конкретном голосовании.                  |
| `voting_info_response`      | Java Kafka Service            | Go API Gateway              | Ответ с подробной информацией о голосовании.                              |
| `get_all_votings_request`   | Go API Gateway                | Java Kafka Service          | Запрос на список всех (или последних) голосований.                        |
| `all_votings_response`      | Java Kafka Service            | Go API Gateway              | Ответ со списком голосований.                                             |
| `blockchain_event_stake`    | (Будущее: Go Event Listener) | Java Kafka Service          | Событие из блокчейна, когда ETH застейкан.                                |
| `blockchain_event_unstake`  | (Будущее: Go Event Listener) | Java Kafka Service          | Событие из блокчейна, когда ETH выведен из стейкинга.                     |
| `blockchain_event_claim`    | (Будущее: Go Event Listener) | Java Kafka Service          | Событие из блокчейна, когда награды получены.                             |

-----

## 9\. Схема базы данных (концептуальная)

Java Kafka Service использует базу данных PostgreSQL. Ниже представлен концептуальный обзор основных таблиц:

  * **`users`**: Хранит информацию о пользователях, в основном их адреса кошельков и, возможно, связанные метаданные.

      * `id` (PK)
      * `wallet_address` (UNIQUE)
      * `creation_date`
      * `last_active`
      * ... (другие поля, специфичные для пользователя)

  * **`votings`**: Хранит детали каждого децентрализованного события голосования.

      * `id` (PK, из события блокчейна или сгенерированный)
      * `title`
      * `description`
      * `creator_id` (FK на `users.id` или `creator_address`)
      * `start_date`
      * `end_date`
      * `is_private` (boolean)
      * `min_votes` (порог для действительности)
      * `creation_date`
      * ... (другие поля, специфичные для голосования)

  * **`voting_options`**: Хранит доступные варианты для каждого голосования.

      * `option_id` (PK)
      * `voting_id` (FK на `votings.id`)
      * `text` (например, "Да", "Нет", "Вариант А")

  * **`votes`**: Записывает каждый отдельный поданный голос.

      * `id` (PK)
      * `voting_id` (FK на `votings.id`)
      * `voter_id` (FK на `users.id` или `voter_address`)
      * `option_id` (FK на `voting_options.id`)
      * `vote_date`
      * `transaction_hash` (опционально, для доказательства в цепочке)

-----

## 10\. Устранение неполадок

  * **Искаженные русские символы в Java-логах:**
      * **Симптом:** Логи показывают `╨Я╨╛╨╗╤Г╤З╨╡╨╜` вместо русского текста.
      * **Решение:** Это проблема кодировки.
          * **Java-приложение:** Убедитесь, что `application.properties` включает `server.servlet.encoding.charset=UTF-8`, `logging.charset.console=UTF-8`, `logging.charset.file=UTF-8`.
          * **Аргументы JVM:** При запуске `java -jar` попробуйте добавить `-Dfile.encoding=UTF-8`.
          * **Терминал/IDE:** Настройте эмулятор терминала или консоль IDE на использование кодировки UTF-8.
  * **`IllegalArgumentException: Пользователь не найден: 0x...` в Java-логах:**
      * **Симптом:** При голосовании Java-сервис жалуется, что адрес пользователя не существует.
      * **Причина:** Бэкенд ожидает, что пользователь (адрес кошелька) будет существовать в его базе данных до обработки действия, связанного с ним (например, голосования).
      * **Решение:** Реализуйте **автоматическое создание пользователя** в `VoteCastHandler` вашего Java Kafka Service (и, возможно, в других обработчиках, таких как `StakeHandler`, если вы храните данные стейкинга для каждого пользователя). Если пользователь с данным адресом кошелька не найден, создайте новую запись `User` в базе данных.
  * **`id: <nil>` в Java-логах для голосований:**
      * **Симптом:** При получении списка голосований одна или несколько записей показывают `id: <nil>`.
      * **Причина:** Это указывает на проблему целостности данных, при которой записи голосования либо не был присвоен правильный ID при создании, либо существует проблема маппинга в вашей Hibernate-сущности.
      * **Решение:**
        1.  **Проверьте сущность `Voting`:** Убедитесь, что аннотации `@Id` и `@GeneratedValue` правильно применены для автоинкрементных ID, если это ваша стратегия.
        2.  **Изучите логику создания:** Просмотрите код, отвечающий за сохранение новых голосований в базе данных в вашем Java-сервисе. Убедитесь, что ID правильно устанавливается или обрабатывается после сохранения.
        3.  **Состояние базы данных:** Проверьте вашу таблицу PostgreSQL `votings` на наличие строк с `NULL` или неверными ID.
  * **`Failed to connect to Ethereum client` / `Failed to instantiate StakeManager contract` (Go API Gateway):**
      * **Симптом:** Go-сервис не может запуститься или взаимодействовать с блокчейном.
      * **Причина:** RPC-нода Ethereum (Anvil) не запущена, или её адрес/порт в `config.yaml` неверен. Адрес контракта также может быть неправильным.
      * **Решение:**
        1.  Убедитесь, что `anvil` запущен на `http://localhost:8545`.
        2.  Проверьте `blockchain.rpc_url` в `config.yaml`.
        3.  Дважды проверьте `blockchain.stake_manager_contract_address` с фактическим адресом развернутого контракта.
  * **`Failed to send GetTokens transaction: ... NothingToClaim` / `CooldownClaimNotReached` / `NotEnoughBalanceOnContract` (Фронтенд/Go API Gateway):**
      * **Симптом:** Получение наград завершается сбоем со специфическими ошибками контракта.
      * **Причина:** Это сообщения `revert` из вашего контракта `StakeManager.sol`.
          * `NothingToClaim`: У пользователя нет накопленных наград.
          * `CooldownClaimNotReached`: У пользователя есть награды, но он не может их получить из-за периода охлаждения.
          * `NotEnoughBalanceOnContract`: Сам контракт `StakeManager` не имеет достаточного количества `RewardToken` для распределения.
      * **Решение:**
        1.  **Для `NotEnoughBalanceOnContract`:** Отправьте `RewardToken` на адрес вашего контракта `StakeManager` после развертывания.
        2.  **Для остальных:** Это ожидаемое поведение контракта; убедитесь, что ваш UI предоставляет соответствующую обратную связь.

-----

## 11\. Будущие улучшения

  * **Интеграция голосования в цепочке:** Полностью интегрировать `Voting.sol` с DApp для неизменяемых записей голосования в цепочке.
  * **Подробная панель стейкинга:** Отображение застейканной суммы, накопленных наград и истории получения на странице профиля.
  * **Обновления в реальном времени:** Реализовать WebSocket-соединения для обновлений в реальном времени статуса голосования, новых голосов и активности пользователей.
  * **Панель администратора:** Для управления голосованиями, распределением наград и параметрами контракта.
  * **Набор для тестирования:** Комплексные модульные, интеграционные и сквозные тесты для всех компонентов.
  * **Аудиты безопасности:** Критически важны для DApp, работающего со средствами пользователей.
  * **Оркестрация контейнеров:** Использование Kubernetes для развертывания микросервисов в производственной среде.
  * **Документация API:** Создание документации OpenAPI (Swagger) для API Gateway.
  * **Фронтенд-фреймворк:** Использование современного фронтенд-фреймворка, такого как React, Vue или Angular, для лучшего управления состоянием и повторного использования компонентов.
  * **Слушатели событий смарт-контрактов в Go:** Реализовать слушателей событий в Go API Gateway для непосредственного потребления событий блокчейна и их публикации в Kafka, уменьшая зависимость от ручного опроса или внешних инструментов.

-----

## 12\. Участие в разработке

Мы приветствуем ваш вклад\! Пожалуйста, следуйте этим шагам для участия в разработке:

1.  Сделайте форк репозитория.
2.  Создайте новую ветку (`git checkout -b feature/ваша-новая-фича`).
3.  Внесите свои изменения.
4.  Зафиксируйте свои изменения (`git commit -m 'Добавлена новая фича'`).
5.  Отправьте изменения в свою ветку (`git push origin feature/ваша-новая-фича`).
6.  Создайте Pull Request.
