document.addEventListener('DOMContentLoaded', function() {
    const connectWalletButton = document.getElementById('connectWallet');
    const profileInfo = document.getElementById('profileInfo');
    const userWalletAddress = document.getElementById('userWalletAddress');
    const createdVotingsCount = document.getElementById('createdVotingsCount');
    const participatedVotingsCount = document.getElementById('participatedVotingsCount');
    const userVotingsTableBody = document.getElementById('userVotingsTableBody');

    // Make fetchUserData globally accessible (optional, but convenient for app.js)
    window.fetchUserData = fetchUserData; // <--- СДЕЛАЕМ ГЛОБАЛЬНО ДОСТУПНОЙ

    connectWalletButton.addEventListener('click', async () => {
        if (typeof window.ethereum !== 'undefined') {
            try {
                const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
                const userAddress = accounts[0];
                localStorage.setItem('userAddress', userAddress);
                displayProfile(userAddress);
                fetchUserData(userAddress);
            } catch (error) {
                console.error('User denied account access or other error:', error);
                alert('Не удалось подключить MetaMask. Пожалуйста, разрешите подключение.');
            }
        } else {
            alert('MetaMask не установлен. Пожалуйста, установите его для использования этой функции.');
        }
    });

    const displayProfile = (address) => {
        userWalletAddress.textContent = address;
        profileInfo.style.display = 'block';
        connectWalletButton.style.display = 'none';
    };

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

    // Функция для рендеринга таблицы с голосованиями пользователя
    const renderUserVotingsTable = (votings) => {
        userVotingsTableBody.innerHTML = ''; // Очищаем таблицу

        if (!votings || votings.length === 0) {
            userVotingsTableBody.innerHTML = `<tr><td colspan="5" class="no-votings">Вы пока не создали или не участвовали в голосованиях.</td></tr>`; // colspan увеличено
            return;
        }

        votings.forEach(voting => {
            const row = document.createElement('tr');
            const now = new Date();
            const startDate = new Date(voting.start_date); // <--- Дата начала
            const endDate = new Date(voting.end_date);

            let statusText = '';
            let statusClass = '';

            if (now < startDate) {
                statusText = 'Предстоящее';
                statusClass = 'status-upcoming';
            } else if (now > endDate) {
                statusText = 'Закончено';
                statusClass = 'status-finished';
            } else {
                statusText = 'Активное';
                statusClass = 'status-active';
            }

            let userVerdictText = 'Не голосовал';
            if (voting.user_vote !== undefined && voting.user_vote !== null) {
                userVerdictText = `Вариант ${voting.user_vote + 1}`;
            }

            const votesCount = voting.votes_count || 0;

            row.innerHTML = `
                <td>${voting.title}</td>
                <td>${votesCount}</td>
                <td>${new Date(voting.start_date).toLocaleString()}</td> <td>${userVerdictText}</td>
                <td class="${statusClass}">${statusText}</td>
            `;
            userVotingsTableBody.appendChild(row);
        });
    };

    // Проверка статуса кошелька при загрузке страницы
    const storedAddress = localStorage.getItem('userAddress');
    if (storedAddress) {
        displayProfile(storedAddress);
        fetchUserData(storedAddress);
    }
});