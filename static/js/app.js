document.addEventListener('DOMContentLoaded', function() {
    const votingsList = document.getElementById('votingsList');
    const createButton = document.getElementById('createButton');
    const createModal = document.getElementById('createModal');
    const createCloseButton = document.querySelector('.create-close');
    const cancelCreateButton = document.getElementById('cancelCreate');
    const submitVotingButton = document.getElementById('submitVoting');
    const addOptionButton = document.getElementById('addOption');
    const optionsContainer = document.getElementById('optionsContainer');

    // Новое поле для даты начала
    const startDateInput = document.getElementById('startDate'); // <--- ДОБАВЛЕНО
    const endDateInput = document.getElementById('endDate'); // Получаем и это поле

    // Элементы нового модального окна деталей голосования
    const votingDetailsModal = document.getElementById('votingDetailsModal');
    const detailsCloseButton = document.querySelector('.details-close');
    const closeDetailsModalButton = document.getElementById('closeDetailsModal');
    const modalVotingTitle = document.getElementById('modalVotingTitle');
    const modalVotingDescription = document.getElementById('modalVotingDescription');
    const modalCreatorAddress = document.getElementById('modalCreatorAddress');
    const modalEndDate = document.getElementById('modalEndDate');
    const modalStartDate = document.getElementById('modalStartDate'); // <--- ДОБАВЛЕНО
    const modalStatus = document.getElementById('modalStatus');
    const modalVotesCount = document.getElementById('modalVotesCount');
    const modalVotingOptions = document.getElementById('modalVotingOptions');
    const submitVoteButton = document.getElementById('submitVoteButton');
    const voteMessage = document.getElementById('voteMessage');
    const voteError = document.getElementById('voteError');

    let currentVotingId = null;

    // --- Функции для модального окна создания голосования ---
    createButton.addEventListener('click', () => {
        createModal.style.display = 'block';
        resetCreateForm();
    });

    createCloseButton.addEventListener('click', () => {
        createModal.style.display = 'none';
    });

    cancelCreateButton.addEventListener('click', () => {
        createModal.style.display = 'none';
    });

    window.addEventListener('click', (event) => {
        if (event.target === createModal) {
            createModal.style.display = 'none';
        }
        if (event.target === votingDetailsModal) {
            votingDetailsModal.style.display = 'none';
        }
    });

    addOptionButton.addEventListener('click', () => {
        if (optionsContainer.children.length < 4) {
            const input = document.createElement('input');
            input.type = 'text';
            input.className = 'vote-option';
            input.placeholder = `Вариант ${optionsContainer.children.length + 1}`;
            input.maxLength = 100;
            optionsContainer.appendChild(input);
        } else {
            alert('Максимальное количество вариантов - 4.');
        }
    });

    function resetCreateForm() {
        document.getElementById('voteTitle').value = '';
        document.getElementById('voteDescription').value = '';
        document.querySelector('input[name="voteType"][value="public"]').checked = true;
        document.getElementById('minVotes').value = 1;
        // Установка текущей даты и времени по умолчанию для startDate
        const now = new Date();
        // Форматируем для input type="datetime-local" (YYYY-MM-DDTHH:mm)
        const year = now.getFullYear();
        const month = (now.getMonth() + 1).toString().padStart(2, '0');
        const day = now.getDate().toString().padStart(2, '0');
        const hours = now.getHours().toString().padStart(2, '0');
        const minutes = now.getMinutes().toString().padStart(2, '0');
        const formattedNow = `${year}-${month}-${day}T${hours}:${minutes}`;
        startDateInput.value = formattedNow; // <--- Установка значения по умолчанию

        document.getElementById('endDate').value = ''; // Дата окончания остается пустой
        optionsContainer.innerHTML = `
            <input type="text" class="vote-option" placeholder="Вариант 1" maxlength="100">
            <input type="text" class="vote-option" placeholder="Вариант 2" maxlength="100">
        `;
    }

    function validateVoting(data) {
        if (!data.title.trim()) {
            alert('Пожалуйста, введите название голосования.');
            return false;
        }
        if (data.options.length < 2) {
            alert('Должно быть как минимум 2 варианта ответа.');
            return false;
        }
        for (const option of data.options) {
            if (!option.trim()) {
                alert('Все варианты ответов должны быть заполнены.');
                return false;
            }
        }
        if (!data.start_date) { // <--- Проверка наличия даты начала
            alert('Пожалуйста, укажите дату начала голосования.');
            return false;
        }
        if (!data.end_date) {
            alert('Пожалуйста, укажите дату окончания голосования.');
            return false;
        }

        const startDate = new Date(data.start_date); // <--- Используем дату начала
        const endDate = new Date(data.end_date);

        if (endDate <= startDate) { // <--- Сравнение даты окончания с датой начала
            alert('Дата окончания голосования должна быть позже даты начала.');
            return false;
        }
        if (data.min_votes <= 0) {
            alert('Минимальное количество голосов должно быть больше 0.');
            return false;
        }
        return true;
    }

    // Отправка формы создания голосования
    submitVotingButton.addEventListener('click', async () => {
        const userAddress = localStorage.getItem('userAddress');
        if (!userAddress) {
            alert('Для создания голосования необходимо подключить MetaMask кошелек. Перейдите в Профиль.');
            return;
        }

        const votingData = {
            title: document.getElementById('voteTitle').value,
            description: document.getElementById('voteDescription').value,
            is_private: document.querySelector('input[name="voteType"]:checked').value === 'private',
            min_votes: parseInt(document.getElementById('minVotes').value),
            start_date: new Date(startDateInput.value).toISOString(), // <--- ОТПРАВЛЯЕМ ДАТУ НАЧАЛА
            end_date: new Date(endDateInput.value).toISOString(),     // <--- И ДАТУ ОКОНЧАНИЯ
            options: Array.from(document.querySelectorAll('#optionsContainer .vote-option'))
                .map(input => input.value)
                .filter(text => text.trim() !== ''),
            creator_address: userAddress
        };

        if (!validateVoting(votingData)) return;

        try {
            const response = await fetch('/voting', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(votingData)
            });

            if (response.ok) {
                const result = await response.json();
                alert(`Голосование создано! ID: ${result.voting_id}`);
                createModal.style.display = 'none';
                loadVotings();
            } else {
                const errorText = await response.text();
                console.error('Ошибка от сервера:', errorText);
                alert('Ошибка при создании голосования: ' + errorText);
            }
        } catch (error) {
            console.error('Error:', error);
            alert('Ошибка при создании голосования');
        }
    });

    // --- Функции для отображения голосований и открытия модального окна деталей ---

    async function loadVotings() {
        try {
            const response = await fetch('/voting');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const votings = await response.json();
            renderVotings(votings);
        } catch (error) {
            console.error('Ошибка при загрузке голосований:', error);
            votingsList.innerHTML = '<p class="no-votings">Не удалось загрузить голосования. Попробуйте позже.</p>';
        }
    }

    function renderVotings(votings) {
        votingsList.innerHTML = '';
        if (votings.length === 0) {
            votingsList.innerHTML = '<p class="no-votings">Пока нет активных голосований.</p>';
            return;
        }

        votings.forEach(voting => {
            const votingCard = document.createElement('div');
            votingCard.className = 'voting-card';
            votingCard.dataset.votingId = voting.voting_id;

            const now = new Date();
            const startDate = new Date(voting.start_date); // <--- НОВАЯ ДАТА НАЧАЛА
            const endDate = new Date(voting.end_date);

            let statusText = '';
            let statusClass = '';
            let canVote = false;

            if (now < startDate) {
                statusText = 'Предстоящее';
                statusClass = 'status-upcoming'; // <--- НОВЫЙ КЛАСС ДЛЯ СТИЛЕЙ
            } else if (now > endDate) {
                statusText = 'Закончено';
                statusClass = 'status-finished';
            } else {
                statusText = 'Активное';
                statusClass = 'status-active';
                canVote = true; // Можно голосовать только если активно
            }

            votingCard.innerHTML = `
                <h3>${voting.title}</h3>
                <p>${voting.description}</p>
                <div class="voting-meta">
                    <span>Начало: ${new Date(voting.start_date).toLocaleString()}</span><br> <span>Окончание: ${new Date(voting.end_date).toLocaleString()}</span>
                    <span class="${statusClass}">${statusText}</span>
                </div>
            `;
            votingsList.appendChild(votingCard);

            votingCard.addEventListener('click', () => openVotingDetails(voting.voting_id));
        });
    }

    // Функция для открытия модального окна деталей голосования
    async function openVotingDetails(votingId) {
        currentVotingId = votingId;
        voteMessage.style.display = 'none';
        voteError.style.display = 'none';

        try {
            const response = await fetch(`/voting/${votingId}`);
            if (!response.ok) {
                throw new Error(`Failed to fetch voting details: ${response.statusText}`);
            }
            const voting = await response.json();

            modalVotingTitle.textContent = voting.title;
            modalVotingDescription.textContent = voting.description;
            modalCreatorAddress.textContent = voting.creator_address;
            modalStartDate.textContent = new Date(voting.start_date).toLocaleString(); // <--- Отображаем дату начала
            modalEndDate.textContent = new Date(voting.end_date).toLocaleString();
            modalVotesCount.textContent = voting.votes_count;

            const now = new Date();
            const startDate = new Date(voting.start_date); // <--- ДАТА НАЧАЛА
            const endDate = new Date(voting.end_date);

            let isFinished = now > endDate;
            let isUpcoming = now < startDate; // <--- Проверка на предстоящее
            let isActive = !isFinished && !isUpcoming; // <--- Проверка на активное

            modalStatus.textContent = isUpcoming ? 'Предстоящее' : (isFinished ? 'Закончено' : 'Активное');
            modalStatus.className = isUpcoming ? 'status-upcoming' : (isFinished ? 'status-finished' : 'status-active');

            modalVotingOptions.innerHTML = '';
            voting.options.forEach((option, index) => {
                const optionDiv = document.createElement('div');
                optionDiv.className = 'vote-option-item';
                // Отключаем радио-кнопки и возможность выбора, если голосование неактивно
                optionDiv.innerHTML = `
                    <input type="radio" name="voteOption" value="${index}" id="option${index}" ${!isActive ? 'disabled' : ''}>
                    <label for="option${index}">${option}</label>
                `;
                modalVotingOptions.appendChild(optionDiv);

                if (!isActive) {
                    optionDiv.classList.add('disabled');
                }

                optionDiv.addEventListener('click', () => {
                    if (isActive) { // Только если голосование активно
                        document.querySelectorAll('.vote-option-item').forEach(item => item.classList.remove('selected'));
                        optionDiv.classList.add('selected');
                        optionDiv.querySelector('input[type="radio"]').checked = true;
                    }
                });
            });

            // Деактивируем кнопку голосования, если голосование неактивно
            submitVoteButton.disabled = !isActive;
            if (isUpcoming) {
                submitVoteButton.textContent = 'Голосование ещё не началось';
            } else if (isFinished) {
                submitVoteButton.textContent = 'Голосование завершено';
            } else {
                submitVoteButton.textContent = 'Проголосовать';
            }


            votingDetailsModal.style.display = 'block';

        } catch (error) {
            console.error('Ошибка при загрузке деталей голосования:', error);
            alert('Не удалось загрузить детали голосования.');
            votingDetailsModal.style.display = 'none';
        }
    }

    // Закрытие модального окна деталей голосования
    detailsCloseButton.addEventListener('click', () => {
        votingDetailsModal.style.display = 'none';
    });

    closeDetailsModalButton.addEventListener('click', () => {
        votingDetailsModal.style.display = 'none';
    });

    // --- Логика отправки голоса ---
    submitVoteButton.addEventListener('click', async () => {
        const userAddress = localStorage.getItem('userAddress');
        if (!userAddress) {
            alert('Для голосования необходимо подключить MetaMask кошелек. Перейдите в Профиль.');
            votingDetailsModal.style.display = 'none';
            return;
        }

        const selectedOption = document.querySelector('input[name="voteOption"]:checked');
        if (!selectedOption) {
            voteError.textContent = 'Пожалуйста, выберите вариант для голосования.';
            voteError.style.display = 'block';
            voteMessage.style.display = 'none';
            return;
        }

        const voteIndex = parseInt(selectedOption.value);
        const votingId = currentVotingId;

        try {
            const userDataResponse = await fetch('/user-data', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ user_address: userAddress })
            });
            if (!userDataResponse.ok) {
                throw new Error('Failed to fetch user data for vote check.');
            }
            const userData = await userDataResponse.json();
            const participatedVotings = userData.votings.filter(v => v.UserVote !== null).map(v => v.ID); // Получаем ID голосований, в которых пользователь уже проголосовал

            if (participatedVotings.includes(votingId)) {
                voteError.textContent = 'Вы уже проголосовали в этом опросе.';
                voteError.style.display = 'block';
                voteMessage.style.display = 'none';
                submitVoteButton.disabled = true;
                document.querySelectorAll('.vote-option-item input[type="radio"]').forEach(radio => radio.disabled = true);
                return;
            }

        } catch (error) {
            console.error('Error checking user vote status:', error);
            voteError.textContent = 'Ошибка при проверке статуса вашего голоса. Попробуйте позже.';
            voteError.style.display = 'block';
            return;
        }


        try {
            const response = await fetch(`/vote`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    voting_id: votingId,
                    user_address: userAddress,
                    selected_option_index: voteIndex
                })
            });

            if (response.ok) {
                voteMessage.textContent = 'Ваш голос учтен!';
                voteMessage.style.display = 'block';
                voteError.style.display = 'none';
                submitVoteButton.disabled = true;
                document.querySelectorAll('.vote-option-item input[type="radio"]').forEach(radio => radio.disabled = true);

                const currentVotesCount = parseInt(modalVotesCount.textContent);
                modalVotesCount.textContent = currentVotesCount + 1;
                loadVotings(); // Перезагружаем список голосований на главной

                // Чтобы обновить профиль, если пользователь на странице профиля
                if (window.location.pathname === '/profile') {
                    // Используем функцию из profile.js, если она доступна в глобальной области видимости
                    if (typeof fetchUserData === 'function') {
                        const storedAddress = localStorage.getItem('userAddress');
                        if (storedAddress) {
                            fetchUserData(storedAddress); // Вызываем функцию из profile.js
                        }
                    } else {
                        // Как запасной вариант, если fetchUserData не глобальна
                        window.location.reload();
                    }
                }

            } else {
                const errorText = await response.text();
                voteError.textContent = `Ошибка при голосовании: ${errorText}`;
                voteError.style.display = 'block';
                voteMessage.style.display = 'none';
            }
        } catch (error) {
            console.error('Ошибка сети при отправке голоса:', error);
            voteError.textContent = 'Ошибка сети. Попробуйте позже.';
            voteError.style.display = 'block';
            voteMessage.style.display = 'none';
        }
    });

    // Инициализация загрузки голосований при старте
    loadVotings();
});