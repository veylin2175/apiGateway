// profile.js

document.addEventListener('DOMContentLoaded', function() {
    // --- Получение ссылок на DOM-элементы ---
    const connectWalletButton = document.getElementById('connectWallet');
    const profileInfo = document.getElementById('profileInfo');
    const userWalletAddress = document.getElementById('userWalletAddress');
    const disconnectWalletButton = document.getElementById('disconnectWalletButton');
    const stakeButton = document.getElementById('stakeButton'); // Кнопка стейкинга
    const unstakeButton = document.getElementById('unstakeButton');
    const getTokensButton = document.getElementById('getTokensButton');

    // Элементы в шапке для отображения статистики
    const headerWalletAddressSpan = document.querySelector('#profileInfoHeader .wallet-address');
    const headerCreatedCountSpan = document.querySelector('#profileInfoHeader .created-votings-count');
    const headerParticipatedCountSpan = document.querySelector('#profileInfoHeader .participated-votings-count');

    // Элементы для отображения статистики в основном разделе профиля
    const createdVotingsCount = document.getElementById('createdVotingsCount');
    const participatedVotingsCount = document.getElementById('participatedVotingsCount');

    // Элемент для таблицы истории голосований
    const profileHistoryTableBody = document.getElementById('profileHistoryTableBody');


    // Сделать fetchUserData глобально доступной, если необходимо для других скриптов
    // (пока не используется напрямую, но может пригодиться)
    window.fetchUserData = fetchUserData;

    // --- Обработчик для кнопки "Подключить MetaMask" ---
    connectWalletButton.addEventListener('click', async () => {
        if (typeof window.ethereum !== 'undefined') {
            try {
                // Запрашиваем доступ к аккаунтам MetaMask
                const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
                const userAddress = accounts[0]; // Берем первый аккаунт
                localStorage.setItem('userAddress', userAddress); // Сохраняем адрес в localStorage

                // Отображаем профиль и загружаем данные
                displayProfile(userAddress);
                fetchUserDataAndHistory(userAddress);

                // Отправляем событие о подключении кошелька на бэкенд
                sendWalletConnectEventToBackend(userAddress);

            } catch (error) {
                console.error('User denied account access or other error:', error);
                alert('Не удалось подключить MetaMask. Пожалуйста, разрешите подключение.');
            }
        } else {
            alert('MetaMask не установлен. Пожалуйста, установите его для использования этой функции.');
        }
    });

    // --- Обработчик для кнопки "Выйти из аккаунта" ---
    disconnectWalletButton.addEventListener('click', () => {
        disconnectWallet();
    });

    // --- Обработчик для кнопки "Stake ETH" ---
    if (stakeButton) { // Проверяем, что кнопка существует в DOM
        stakeButton.addEventListener('click', stakeEth);
    }

    if (unstakeButton) { // Проверяем, что кнопка существует в DOM
        unstakeButton.addEventListener('click', unstakeEth);
    }

    if (getTokensButton) {
        getTokensButton.addEventListener('click', getTokens);
    }

    // --- Функция для отображения/скрытия разделов профиля ---
    const displayProfile = (address) => {
        if (address) {
            userWalletAddress.textContent = address; // Обновляем адрес в профиле
            profileInfo.style.display = 'block';    // Показываем раздел профиля
            connectWalletButton.style.display = 'none'; // Скрываем кнопку подключения

            // Обновляем информацию в шапке
            if (headerWalletAddressSpan) {
                headerWalletAddressSpan.textContent = address;
            }
            // Сбрасываем счетчики в шапке при подключении (или они будут обновлены позже)
            if (headerCreatedCountSpan) {
                headerCreatedCountSpan.textContent = `Создано: 0`;
            }
            if (headerParticipatedCountSpan) {
                headerParticipatedCountSpan.textContent = `Проголосовал: 0`;
            }
            // Устанавливаем сообщение о загрузке для истории голосований
            if (profileHistoryTableBody) {
                profileHistoryTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">История загружается...</td></tr>`;
            }

        } else { // Если адрес null или пустой, скрываем профиль и показываем кнопку подключения
            userWalletAddress.textContent = '';
            profileInfo.style.display = 'none';
            connectWalletButton.style.display = 'block';

            // Очищаем информацию в шапке
            if (headerWalletAddressSpan) {
                headerWalletAddressSpan.textContent = '';
            }
            if (headerCreatedCountSpan) {
                headerCreatedCountSpan.textContent = `Создано: 0`;
            }
            if (headerParticipatedCountSpan) {
                headerParticipatedCountSpan.textContent = `Проголосовал: 0`;
            }
            // Устанавливаем сообщение для истории, когда кошелек не подключен
            if (profileHistoryTableBody) {
                profileHistoryTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">Для просмотра истории подключите ваш MetaMask кошелек.</td></tr>`;
            }
        }
    };

    // --- Функция для отключения кошелька ---
    const disconnectWallet = () => {
        localStorage.removeItem('userAddress'); // Удаляем адрес из localStorage
        displayProfile(null); // Обновляем UI, показывая состояние "не подключено"
        console.log('Кошелек отключен. Локальное хранилище очищено.');
    };

    // --- Функция для отправки события подключения кошелька на бэкенд ---
    async function sendWalletConnectEventToBackend(walletAddress) {
        try {
            const response = await fetch('/connect-wallet', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ walletAddress: walletAddress })
            });
            const data = await response.text();
            console.log('Ответ бэкенда на подключение кошелька:', data);
        } catch (error) {
            console.error('Ошибка при отправке события подключения кошелька на бэкенд:', error);
        }
    }

    // --- Функция для загрузки данных пользователя с бэкенда ---
    async function fetchUserData(userAddress) {
        try {
            const response = await fetch('/user-data', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ user_address: userAddress })
            });

            if (response.ok) {
                const userData = await response.json();
                console.log('Получены данные пользователя:', userData);

                // Обновляем счетчики голосований в профиле
                createdVotingsCount.textContent = userData.created_votings_count;
                participatedVotingsCount.textContent = userData.participated_votings_count;

                // Обновляем счетчики в шапке
                if (headerCreatedCountSpan) {
                    headerCreatedCountSpan.textContent = `Создано: ${userData.created_votings_count}`;
                }
                if (headerParticipatedCountSpan) {
                    headerParticipatedCountSpan.textContent = `Проголосовал: ${userData.participated_votings_count}`;
                }

                // Рендерим таблицу истории голосований
                renderProfileHistoryTable(userData.history);

            } else {
                const errorText = await response.text();
                console.error('Ошибка при загрузке данных пользователя:', response.status, errorText);
                alert('Не удалось загрузить данные профиля: ' + errorText);
            }
        } catch (error) {
            console.error('Ошибка при получении данных пользователя:', error);
            alert('Ошибка при загрузке данных пользователя.');
        }
    }

    // --- Функция для загрузки данных пользователя и истории (с задержкой для Kafka) ---
    async function fetchUserDataAndHistory(userAddress) {
        // Первый вызов для получения основной информации
        await fetchUserData(userAddress);

        // Повторный вызов через несколько секунд для получения обновленной истории
        // (даем время Kafka и Java-сервису обработать запрос)
        setTimeout(async () => {
            console.log("Повторный запрос данных пользователя для обновления истории...");
            await fetchUserData(userAddress); // Повторный вызов
        }, 2500); // 2.5 секунды - можно настроить
    }

    // --- Функция для рендеринга таблицы истории голосований ---
    const renderProfileHistoryTable = (history) => {
        if (!profileHistoryTableBody) {
            console.warn('Элемент #profileHistoryTableBody не найден в DOM.');
            return;
        }

        profileHistoryTableBody.innerHTML = ''; // Очищаем таблицу перед заполнением

        if (!history || history.length === 0) {
            profileHistoryTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">История голосований не найдена.</td></tr>`;
            return;
        }

        history.forEach(entry => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${entry.title || 'Без названия'}</td>
                <td>${entry.votersCount || 0}</td>
                <td>${entry.isPrivate ? 'Да' : 'Нет'}</td>
                <td>${entry.optionText || 'Не указан'}</td>
            `;
            profileHistoryTableBody.appendChild(row);
        });
    };

    // --- НОВАЯ ФУНКЦИЯ: Обработчик стейкинга ---
    async function stakeEth() {
        // ИСПРАВЛЕНИЕ: Используем userWalletAddress для получения текста адреса
        const walletAddress = userWalletAddress.textContent;

        if (!walletAddress || walletAddress === 'Адрес кошелька: Not Connected' || walletAddress === '') {
            alert('Пожалуйста, подключите свой кошелек, чтобы внести ETH.');
            return;
        }

        const amountStr = prompt('Введите сумму для стейкинга (в ETH):');
        if (!amountStr || isNaN(amountStr) || parseFloat(amountStr) <= 0) {
            alert('Пожалуйста, введите корректную положительную сумму.');
            return;
        }
        const amount = parseFloat(amountStr); // Сумма в ETH

        console.log(`Попытка стейкинга ${amount} ETH с адреса ${walletAddress}`);

        try {
            const response = await fetch('/staking', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    amount: amount,
                    staker_address: walletAddress // Отправляем адрес стейкера на бэкенд
                })
            });

            const data = await response.json();

            if (response.ok) {
                alert(`Стейкинг успешно выполнен! Хеш транзакции: ${data.tx_hash}`);
                console.log('Стейкинг успешно выполнен:', data);
                // После успешного стейкинга, обновите данные пользователя
                // чтобы, например, отобразить обновленный баланс или историю
                fetchUserDataAndHistory(walletAddress);
            } else {
                alert(`Стейкинг не удался: ${data.error || 'Неизвестная ошибка'}`);
                console.error('Стейкинг не удался:', data);
            }
        } catch (error) {
            console.error('Ошибка во время запроса стейкинга:', error);
            alert('Произошла ошибка во время стейкинга. Проверьте консоль.');
        }
    }

    async function unstakeEth() {
        const walletAddress = userWalletAddress.textContent;

        if (!walletAddress || walletAddress === 'Адрес кошелька: Not Connected' || walletAddress === '') {
            alert('Пожалуйста, подключите свой кошелек, чтобы вывести ETH.');
            return;
        }

        if (!confirm('Вы уверены, что хотите вывести весь застейканный ETH?')) {
            return; // Пользователь отменил операцию
        }

        console.log(`Попытка вывода ETH с адреса ${walletAddress}`);

        try {
            const response = await fetch('/unstake', { // НОВЫЙ ЭНДПОИНТ
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    staker_address: walletAddress // Только адрес стейкера
                })
            });

            const data = await response.json();

            if (response.ok) {
                alert(`Вывод ETH успешно выполнен! Хеш транзакции: ${data.tx_hash}`);
                console.log('Вывод ETH успешно выполнен:', data);
                // Обновляем данные профиля, чтобы отобразить изменения баланса
                fetchUserDataAndHistory(walletAddress);
            } else {
                alert(`Вывод ETH не удался: ${data.error || 'Неизвестная ошибка'}`);
                console.error('Вывод ETH не удался:', data);
            }
        } catch (error) {
            console.error('Ошибка во время запроса вывода ETH:', error);
            alert('Произошла ошибка во время вывода ETH. Проверьте консоль.');
        }
    }

    async function getTokens() {
        const walletAddress = userWalletAddress.textContent;

        if (!walletAddress || walletAddress === 'Адрес кошелька: Not Connected' || walletAddress === '') {
            alert('Пожалуйста, подключите свой кошелек, чтобы получить награды.');
            return;
        }

        if (!confirm('Вы уверены, что хотите получить доступные награды?')) {
            return; // Пользователь отменил операцию
        }

        console.log(`Попытка получения наград с адреса ${walletAddress}`);

        try {
            // ИСПОЛЬЗУЕМ НОВЫЙ МАРШРУТ /profile/get_tokens
            const response = await fetch('/get_tokens', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                // Для этой функции не требуется body, так как адрес берется из backend'а по приватному ключу
            });

            const data = await response.json();

            if (response.ok) {
                alert(`Награды успешно получены! Хеш транзакции: ${data.data.tx_hash}`); // Обратите внимание на data.data.tx_hash
                console.log('Награды успешно получены:', data);
                // После успешного клейма, обновите данные пользователя (например, баланс токенов)
                fetchUserDataAndHistory(walletAddress);
            } else {
                alert(`Не удалось получить награды: ${data.message || 'Неизвестная ошибка'}`);
                console.error('Не удалось получить награды:', data);
            }
        } catch (error) {
            console.error('Ошибка во время запроса получения наград:', error);
            alert('Произошла ошибка во время получения наград. Проверьте консоль.');
        }
    }

    // ... (stakeEth, unstakeEth - без изменений) ...

    // --- Логика при загрузке страницы: Проверка и подключение кошелька ---
    const storedAddress = localStorage.getItem('userAddress');
    if (storedAddress) {
        displayProfile(storedAddress);
        fetchUserDataAndHistory(storedAddress);
    } else {
        displayProfile(null);
    }

    // --- Обработчик изменения аккаунтов в MetaMask ---
    if (typeof window.ethereum !== 'undefined') {
        window.ethereum.on('accountsChanged', (newAccounts) => {
            if (newAccounts.length > 0) {
                const newAddress = newAccounts[0];
                localStorage.setItem('userAddress', newAddress);
                displayProfile(newAddress);
                fetchUserDataAndHistory(newAddress);
                console.log('MetaMask аккаунт изменен на:', newAddress);
            } else {
                disconnectWallet();
                console.log('MetaMask: Все аккаунты отключены от этого DApp.');
            }
        });

        window.ethereum.on('disconnect', (code, reason) => {
            console.log('MetaMask disconnected:', code, reason);
            disconnectWallet();
        });
    }
});