document.addEventListener('DOMContentLoaded', function() {
    const connectWalletButton = document.getElementById('connectWallet');
    const profileInfo = document.getElementById('profileInfo');
    const userWalletAddress = document.getElementById('userWalletAddress');
    const disconnectWalletButton = document.getElementById('disconnectWalletButton'); // НОВАЯ ПЕРЕМЕННАЯ

    const headerWalletAddressSpan = document.querySelector('#profileInfoHeader .wallet-address');
    const headerCreatedCountSpan = document.querySelector('#profileInfoHeader .created-votings-count');
    const headerParticipatedCountSpan = document.querySelector('#profileInfoHeader .participated-votings-count');

    const createdVotingsCount = document.getElementById('createdVotingsCount');
    const participatedVotingsCount = document.getElementById('participatedVotingsCount');
    const userVotingsTableBody = document.getElementById('userVotingsTableBody');

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
                fetchUserData(userAddress);

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
            // Очищаем таблицу при новом подключении
            userVotingsTableBody.innerHTML = `<tr><td colspan="5" class="no-votings">Загрузка данных...</td></tr>`;

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
            userVotingsTableBody.innerHTML = `<tr><td colspan="5" class="no-votings">Для просмотра профиля подключите ваш MetaMask кошелек.</td></tr>`;
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


    // --- fetchUserData (без изменений) ---
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

                createdVotingsCount.textContent = userData.created_votings_count;
                participatedVotingsCount.textContent = userData.participated_votings_count;

                if (headerCreatedCountSpan) {
                    headerCreatedCountSpan.textContent = `Создано: ${userData.created_votings_count}`;
                }
                if (headerParticipatedCountSpan) {
                    headerParticipatedCountSpan.textContent = `Проголосовал: ${userData.participated_votings_count}`;
                }

                renderUserVotingsTable(userData.votings);
            } else {
                const errorText = await response.text();
                console.error('Ошибка при загрузке данных пользователя:', errorText);
                alert('Не удалось загрузить данные профиля: ' + errorText);
            }
        } catch (error) {
            console.error('Error fetching user data:', error);
            alert('Ошибка при загрузке данных пользователя.');
        }
    }

    // --- renderUserVotingsTable (без изменений) ---
    const renderUserVotingsTable = (votings) => {
        userVotingsTableBody.innerHTML = '';

        if (!votings || votings.length === 0) {
            userVotingsTableBody.innerHTML = `<tr><td colspan="5" class="no-votings">Вы пока не создали или не участвовали в голосованиях.</td></tr>`;
            return;
        }

        votings.forEach(voting => {
            const row = document.createElement('tr');

            let statusText = voting.status;
            let statusClass = '';

            switch (voting.status) {
                case 'Upcoming':
                    statusClass = 'status-upcoming';
                    break;
                case 'Active':
                    statusClass = 'status-active';
                    break;
                case 'Finished':
                    statusClass = 'status-finished';
                    break;
                case 'Rejected':
                    statusClass = 'status-rejected';
                    break;
                default:
                    statusClass = 'status-unknown';
            }

            let userVerdictText = 'Не голосовал';
            if (voting.user_vote !== undefined && voting.user_vote !== null) {
                userVerdictText = `Вариант ${voting.user_vote + 1}`;
            }

            const votesCount = voting.votes_count || 0;
            const votingType = voting.is_private ? 'Приватное' : 'Публичное';

            row.innerHTML = `
            <td>${voting.title}</td>
            <td>${votesCount}</td>
            <td>${votingType}</td>
            <td>${userVerdictText}</td>
            <td class="${statusClass}">${statusText}</td>
        `;
            userVotingsTableBody.appendChild(row);
        });
    };

    // --- Логика при загрузке страницы (без изменений, но использует модифицированную displayProfile) ---
    const storedAddress = localStorage.getItem('userAddress');
    if (storedAddress) {
        displayProfile(storedAddress); // Обновить UI профиля
        fetchUserData(storedAddress); // Загрузить данные с сервера
    }

    // --- Обработчик изменения аккаунтов в MetaMask ---
    if (typeof window.ethereum !== 'undefined') {
        window.ethereum.on('accountsChanged', (newAccounts) => {
            if (newAccounts.length > 0) {
                const newAddress = newAccounts[0];
                localStorage.setItem('userAddress', newAddress);
                displayProfile(newAddress);
                fetchUserData(newAddress);
                console.log('MetaMask account changed to:', newAddress);
            } else {
                // Все аккаунты были отключены от DApp в MetaMask
                disconnectWallet(); // Вызываем нашу функцию отключения
                console.log('MetaMask: All accounts disconnected from this DApp.');
            }
        });
    }
});