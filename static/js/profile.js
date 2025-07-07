document.addEventListener('DOMContentLoaded', async () => {
    const authSection = document.getElementById('authSection');
    const profileDetailsSection = document.getElementById('profileDetails');
    const connectButton = document.getElementById('connectMetamask');
    const metamaskError = document.getElementById('metamaskError');
    const profileInfoHeader = document.getElementById('profileInfoHeader');
    const walletAddressSpan = profileInfoHeader.querySelector('.wallet-address');
    const createdVotingsCountSpan = profileInfoHeader.querySelector('.created-votings-count');
    const participatedVotingsCountSpan = profileInfoHeader.querySelector('.participated-votings-count');
    const userVotingsTableBody = document.getElementById('userVotingsTableBody');

    let currentAccount = null; // Глобальная переменная для хранения текущего адреса кошелька

    // Функция для подключения MetaMask
    const connectMetamask = async () => {
        metamaskError.style.display = 'none';
        if (typeof window.ethereum !== 'undefined') {
            try {
                const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
                if (accounts.length === 0) {
                    throw new Error("Нет доступных аккаунтов. Пожалуйста, создайте или импортируйте аккаунт в MetaMask.");
                }
                currentAccount = accounts[0];
                localStorage.setItem('userAddress', currentAccount); // Сохраняем адрес в localStorage
                await fetchUserData(currentAccount);
                displayProfile(currentAccount);
            } catch (error) {
                console.error("Ошибка подключения MetaMask:", error);
                metamaskError.textContent = `Ошибка: ${error.message || "Не удалось подключиться к MetaMask."}`;
                metamaskError.style.display = 'block';
                authSection.style.display = 'block';
                profileDetailsSection.style.display = 'none';
            }
        } else {
            metamaskError.textContent = "MetaMask не установлен. Пожалуйста, установите MetaMask для использования этого сервиса.";
            metamaskError.style.display = 'block';
            authSection.style.display = 'block';
            profileDetailsSection.style.display = 'none';
        }
    };

    // Функция для отображения профиля
    const displayProfile = (address) => {
        authSection.style.display = 'none';
        profileDetailsSection.style.display = 'block';
        walletAddressSpan.textContent = address;
    };

    // Функция для получения данных пользователя с бэкенда
    const fetchUserData = async (address) => {
        try {
            // Запрос на получение всех голосований (включая приватные, чтобы потом их фильтровать)
            const response = await fetch('/user-data', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ user_address: address })
            });

            if (response.ok) {
                const userData = await response.json();
                console.log("Получены данные пользователя:", userData);
                updateProfileHeader(userData);
                renderUserVotingsTable(userData.votings);
            } else {
                const errorText = await response.text();
                console.error("Ошибка при получении данных пользователя:", errorText);
                metamaskError.textContent = `Ошибка загрузки данных: ${errorText}`;
                metamaskError.style.display = 'block';
                authSection.style.display = 'block';
                profileDetailsSection.style.display = 'none';
            }
        } catch (error) {
            console.error("Ошибка сети при получении данных пользователя:", error);
            metamaskError.textContent = `Ошибка сети: ${error.message}`;
            metamaskError.style.display = 'block';
            authSection.style.display = 'block';
            profileDetailsSection.style.display = 'none';
        }
    };

    // Функция для обновления заголовка профиля
    const updateProfileHeader = (userData) => {
        createdVotingsCountSpan.textContent = `Создано: ${userData.created_votings_count}`;
        participatedVotingsCountSpan.textContent = `Проголосовал: ${userData.participated_votings_count}`;
    };

    // Функция для рендеринга таблицы с голосованиями пользователя
    const renderUserVotingsTable = (votings) => {
        userVotingsTableBody.innerHTML = ''; // Очищаем таблицу

        if (!votings || votings.length === 0) {
            userVotingsTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">Вы пока не создали или не участвовали в голосованиях.</td></tr>`;
            return;
        }

        votings.forEach(voting => {
            const row = document.createElement('tr');
            const now = new Date();
            const endDate = new Date(voting.end_date);
            const statusClass = now > endDate ? 'status-finished' : 'status-active';
            const statusText = now > endDate ? 'Закончено' : 'Активное';

            // Ваш вердикт: если user_vote есть, отображаем название варианта, иначе "Не голосовал"
            let userVerdictText = 'Не голосовал';
            if (voting.user_vote !== undefined && voting.user_vote !== null) {
                // Предполагаем, что у нас есть доступ к options голосования,
                // но в текущей UserVotingDetail их нет. Для MVP, пока просто покажем индекс.
                // В реальной системе нужно либо передавать options в UserVotingDetail,
                // либо делать дополнительный запрос за деталями голосования.
                // Пока используем просто "Вариант N"
                userVerdictText = `Вариант ${voting.user_vote + 1}`; // +1 потому что индексы 0-based
            }

            // Внимание: Если вы хотите показать само название варианта ("Синий", "Да"),
            // то вам нужно будет либо передавать массив options в UserVotingDetail
            // (что усложнит структуру), либо делать дополнительный запрос на /voting/{id}
            // для каждого голосования в профиле, чтобы получить его опции.
            // Для простоты MVP, оставим "Вариант N"

            const votesCount = voting.votes_count || 0;

            row.innerHTML = `
            <td>${voting.title}</td>
            <td>${votesCount}</td>
            <td>${userVerdictText}</td>
            <td class="${statusClass}">${statusText}</td>
        `;
            userVotingsTableBody.appendChild(row);
        });
    };

    // Проверка статуса авторизации при загрузке страницы
    const checkAuthStatus = async () => {
        const storedAddress = localStorage.getItem('userAddress');
        if (storedAddress) {
            // Если адрес есть в localStorage, пытаемся получить аккаунты из MetaMask
            if (typeof window.ethereum !== 'undefined') {
                try {
                    const accounts = await window.ethereum.request({ method: 'eth_accounts' }); // Получаем аккаунты без запроса на подключение
                    if (accounts.length > 0 && accounts[0].toLowerCase() === storedAddress.toLowerCase()) {
                        currentAccount = accounts[0];
                        await fetchUserData(currentAccount);
                        displayProfile(currentAccount);
                        return;
                    } else {
                        // Адрес в localStorage не совпадает с активным аккаунтом MetaMask или нет активных аккаунтов
                        localStorage.removeItem('userAddress'); // Удаляем устаревший адрес
                    }
                } catch (error) {
                    console.error("Ошибка при проверке аккаунтов MetaMask:", error);
                    localStorage.removeItem('userAddress');
                }
            } else {
                localStorage.removeItem('userAddress'); // MetaMask не установлен
            }
        }
        // Если не авторизован или ошибка, показываем кнопку подключения
        authSection.style.display = 'block';
        profileDetailsSection.style.display = 'none';
    };

    // Слушатель для кнопки подключения MetaMask
    connectButton.addEventListener('click', connectMetamask);

    // Слушатель для изменения аккаунта в MetaMask
    if (typeof window.ethereum !== 'undefined') {
        window.ethereum.on('accountsChanged', (accounts) => {
            if (accounts.length === 0) {
                // Пользователь отключил все аккаунты
                localStorage.removeItem('userAddress');
                currentAccount = null;
                authSection.style.display = 'block';
                profileDetailsSection.style.display = 'none';
                walletAddressSpan.textContent = '';
                createdVotingsCountSpan.textContent = `Создано: 0`;
                participatedVotingsCountSpan.textContent = `Проголосовал: 0`;
                userVotingsTableBody.innerHTML = `<tr><td colspan="4" class="no-votings">Вы пока не создали или не участвовали в голосованиях.</td></tr>`;
                console.log("Аккаунт MetaMask отключен.");
            } else {
                // Аккаунт изменен
                currentAccount = accounts[0];
                localStorage.setItem('userAddress', currentAccount);
                fetchUserData(currentAccount); // Перезагружаем данные для нового аккаунта
                displayProfile(currentAccount);
                console.log("Аккаунт MetaMask изменен на:", currentAccount);
            }
        });
        window.ethereum.on('chainChanged', (chainId) => {
            console.log("Сеть MetaMask изменена на:", chainId);
            // Возможно, здесь потребуется перезагрузка данных или предупреждение
            // Для MVP пока просто логируем
        });
    }

    // Инициализация: проверка авторизации при загрузке страницы
    checkAuthStatus();
});