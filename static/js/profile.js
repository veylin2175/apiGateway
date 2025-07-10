// profile.js

document.addEventListener('DOMContentLoaded', function() {
    const connectWalletButton = document.getElementById('connectWallet');
    const profileInfo = document.getElementById('profileInfo');
    const userWalletAddress = document.getElementById('userWalletAddress');
    const disconnectWalletButton = document.getElementById('disconnectWalletButton');

    const headerWalletAddressSpan = document.querySelector('#profileInfoHeader .wallet-address');
    const headerCreatedCountSpan = document.querySelector('#profileInfoHeader .created-votings-count');
    const headerParticipatedCountSpan = document.querySelector('#profileInfoHeader .participated-votings-count');

    const createdVotingsCount = document.getElementById('createdVotingsCount');
    const participatedVotingsCount = document.getElementById('participatedVotingsCount');
    // УДАЛЯЕМ: const userVotingsTableBody = document.getElementById('userVotingsTableBody');

    // НОВАЯ ПЕРЕМЕННАЯ: tbody для таблицы истории голосований
    const profileHistoryTableBody = document.getElementById('profileHistoryTableBody');


    // Make fetchUserData globally accessible for app.js (пока не используется, но оставляем)
    window.fetchUserData = fetchUserData;

    // --- Обработчик для кнопки "Подключить MetaMask" (без изменений) ---
    connectWalletButton.addEventListener('click', async () => {
        if (typeof window.ethereum !== 'undefined') {
            try {
                const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
                const userAddress = accounts[0];
                localStorage.setItem('userAddress', userAddress);
                displayProfile(userAddress);
                fetchUserDataAndHistory(userAddress); // ИСПОЛЬЗУЕМ НОВУЮ ФУНКЦИЮ

                // Опционально: отправить событие регистрации на бэкенд
                sendWalletConnectEventToBackend(userAddress);

            } catch (error) {
                console.error('User denied account access or other error:', error);
                alert('Не удалось подключить MetaMask. Пожалуйста, разрешите подключение.');
            }
        } else {
            alert('MetaMask не установлен. Пожалуйста, установите его для использования этой функции.');
        }
    });

    // --- НОВЫЙ ОБРАБОТЧИК: для кнопки "Выйти из аккаунта" ---
    disconnectWalletButton.addEventListener('click', () => {
        disconnectWallet();
    });

    // --- Модифицированная функция displayProfile ---
    const displayProfile = (address) => {
        if (address) {
            userWalletAddress.textContent = address;
            profileInfo.style.display = 'block';
            connectWalletButton.style.display = 'none';

            // Обновляем информацию в шапке
            if (headerWalletAddressSpan) {
                headerWalletAddressSpan.textContent = address;
            }
            // Сбрасываем счетчики в шапке при подключении (или они будут обновлены fetchUserData)
            if (headerCreatedCountSpan) {
                headerCreatedCountSpan.textContent = `Создано: 0`;
            }
            if (headerParticipatedCountSpan) {
                headerParticipatedCountSpan.textContent = `Проголосовал: 0`;
            }
            // УДАЛЯЕМ: userVotingsTableBody.innerHTML = `<tr><td colspan="5" class="no-votings">Загрузка данных...</td></tr>`;

            // Очищаем и устанавливаем сообщение о загрузке для истории
            if (profileHistoryTableBody) {
                profileHistoryTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">История загружается...</td></tr>`;
            }

        } else { // Если адрес null, то скрываем профиль и показываем кнопку подключения
            userWalletAddress.textContent = '';
            profileInfo.style.display = 'none';
            connectWalletButton.style.display = 'block';

            // Очищаем информацию в шапке при отключении
            if (headerWalletAddressSpan) {
                headerWalletAddressSpan.textContent = '';
            }
            if (headerCreatedCountSpan) {
                headerCreatedCountSpan.textContent = `Создано: 0`;
            }
            if (headerParticipatedCountSpan) {
                headerParticipatedCountSpan.textContent = `Проголосовал: 0`;
            }
            // УДАЛЯЕМ: userVotingsTableBody.innerHTML = `<tr><td colspan="5" class="no-votings">Для просмотра профиля подключите ваш MetaMask кошелек.</td></tr>`;
            if (profileHistoryTableBody) {
                profileHistoryTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">Для просмотра истории подключите ваш MetaMask кошелек.</td></tr>`;
            }
        }
    };

    // --- НОВАЯ ФУНКЦИЯ: disconnectWallet ---
    const disconnectWallet = () => {
        localStorage.removeItem('userAddress'); // Удаляем адрес кошелька из localStorage
        displayProfile(null); // Обновляем UI, показывая состояние "не подключено"
        console.log('Wallet disconnected. Local storage cleared.');

        // Опционально: можно перенаправить пользователя на главную страницу
        // window.location.href = '/';
    };

    // --- Модифицированная функция sendWalletConnectEventToBackend ---
    // Переименована для ясности и сделана локальной
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
            console.log('Backend response for connect-wallet:', data);
        } catch (error) {
            console.error('Error sending wallet connect event to backend:', error);
        }
    }


    // --- fetchUserData: ОБНОВЛЕННАЯ ФУНКЦИЯ ---
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
                console.log('Received user data:', userData); // Проверяем полный ответ

                createdVotingsCount.textContent = userData.created_votings_count;
                participatedVotingsCount.textContent = userData.participated_votings_count;

                if (headerCreatedCountSpan) {
                    headerCreatedCountSpan.textContent = `Создано: ${userData.created_votings_count}`;
                }
                if (headerParticipatedCountSpan) {
                    headerParticipatedCountSpan.textContent = `Проголосовал: ${userData.participated_votings_count}`;
                }

                // УДАЛЯЕМ: renderUserVotingsTable(userData.votings); // Рендерим таблицу созданных/участвующих голосований

                // --- Рендеринг таблицы истории голосований ---
                renderProfileHistoryTable(userData.history);
                // --- КОНЕЦ БЛОКА ---

            } else {
                const errorText = await response.text();
                console.error('Ошибка при загрузке данных пользователя:', response.status, errorText);
                alert('Не удалось загрузить данные профиля: ' + errorText);
            }
        } catch (error) {
            console.error('Error fetching user data:', error);
            alert('Ошибка при загрузке данных пользователя.');
        }
    }

    // --- НОВАЯ ФУНКЦИЯ: fetchUserDataAndHistory ---
    // Эта функция будет вызывать fetchUserData и затем планировать повторный опрос
    async function fetchUserDataAndHistory(userAddress) {
        // Первый вызов для получения основной информации и инициирования запроса истории
        await fetchUserData(userAddress);

        // Повторный вызов через несколько секунд для получения обновленной истории
        // Даём время Kafka и Java-сервису обработать запрос
        setTimeout(async () => {
            console.log("Повторный запрос данных пользователя для обновления истории...");
            await fetchUserData(userAddress); // Повторный вызов
        }, 2500); // 2.5 секунды - можно настроить
    }

    // --- НОВАЯ ФУНКЦИЯ: renderProfileHistoryTable ---
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


    // --- Логика при загрузке страницы: ИСПОЛЬЗУЕМ НОВУЮ ФУНКЦИЮ fetchUserDataAndHistory ---
    const storedAddress = localStorage.getItem('userAddress');
    if (storedAddress) {
        displayProfile(storedAddress); // Обновить UI профиля
        fetchUserDataAndHistory(storedAddress); // Загрузить данные с сервера и инициировать историю
    }

    // --- Обработчик изменения аккаунтов в MetaMask ---
    if (typeof window.ethereum !== 'undefined') {
        window.ethereum.on('accountsChanged', (newAccounts) => {
            if (newAccounts.length > 0) {
                const newAddress = newAccounts[0];
                localStorage.setItem('userAddress', newAddress);
                displayProfile(newAddress);
                fetchUserDataAndHistory(newAddress); // ИСПОЛЬЗУЕМ НОВУЮ ФУНКЦИЮ
                console.log('MetaMask account changed to:', newAddress);
            } else {
                // Все аккаунты были отключены от DApp в MetaMask
                disconnectWallet(); // Вызываем нашу функцию отключения
                console.log('MetaMask: All accounts disconnected from this DApp.');
            }
        });
    }
});