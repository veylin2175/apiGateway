document.addEventListener('DOMContentLoaded', function() {
    const connectWalletButton = document.getElementById('connectWallet');
    const profileInfo = document.getElementById('profileInfo');
    const userWalletAddress = document.getElementById('userWalletAddress');

    const headerWalletAddressSpan = document.querySelector('#profileInfoHeader .wallet-address');
    const headerCreatedCountSpan = document.querySelector('#profileInfoHeader .created-votings-count');
    const headerParticipatedCountSpan = document.querySelector('#profileInfoHeader .participated-votings-count');

    const createdVotingsCount = document.getElementById('createdVotingsCount');
    const participatedVotingsCount = document.getElementById('participatedVotingsCount');
    const userVotingsTableBody = document.getElementById('userVotingsTableBody');

    // Make fetchUserData globally accessible for app.js
    window.fetchUserData = fetchUserData;

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

        if (headerWalletAddressSpan) {
            headerWalletAddressSpan.textContent = address;
        }
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

    // Исправление ошибки с голосованием после перезахода:
    // При загрузке страницы, если адрес кошелька сохранен,
    // сразу же пытаемся загрузить данные пользователя.
    const storedAddress = localStorage.getItem('userAddress');
    if (storedAddress) {
        displayProfile(storedAddress); // Обновить UI профиля
        fetchUserData(storedAddress); // Загрузить данные с сервера
    }
});