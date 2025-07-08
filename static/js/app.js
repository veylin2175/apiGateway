document.addEventListener('DOMContentLoaded', function() {
    const votingsList = document.getElementById('votingsList');
    const createButton = document.getElementById('createButton');
    const createModal = document.getElementById('createModal');
    const createCloseButton = document.querySelector('.create-close');
    const cancelCreateButton = document.getElementById('cancelCreate');
    const submitVotingButton = document.getElementById('submitVoting');
    const addOptionButton = document.getElementById('addOption');
    const optionsContainer = document.getElementById('optionsContainer');

    const startDateInput = document.getElementById('startDate');
    const endDateInput = document.getElementById('endDate');

    const votingDetailsModal = document.getElementById('votingDetailsModal');
    const detailsCloseButton = document.querySelector('.details-close'); // Убедитесь, что этот селектор верный для кнопки закрытия в деталях
    const closeDetailsModalButton = document.getElementById('closeDetailsModal'); // Добавлена кнопка "Закрыть" в футере модалки

    const modalVotingTitle = document.getElementById('modalVotingTitle');
    const modalVotingDescription = document.getElementById('modalVotingDescription');
    const modalCreatorAddress = document.getElementById('modalCreatorAddress');
    const modalEndDate = document.getElementById('modalEndDate');
    const modalStartDate = document.getElementById('modalStartDate');
    const modalStatus = document.getElementById('modalStatus');
    const modalVotesCount = document.getElementById('modalVotesCount');
    const modalVotingOptions = document.getElementById('modalVotingOptions');
    const submitVoteButton = document.getElementById('submitVoteButton');
    const voteMessage = document.getElementById('voteMessage');
    const voteError = document.getElementById('voteError');
    const modalWinner = document.getElementById('modalWinner');
    const modalMinVotes = document.getElementById('modalMinVotes');

    let currentVotingId = null; // Переменная для хранения ID текущего открытого голосования

    // --- Create Modal Logic ---
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

        const now = new Date();
        const year = now.getFullYear();
        const month = (now.getMonth() + 1).toString().padStart(2, '0');
        const day = now.getDate().toString().padStart(2, '0');
        const hours = now.getHours().toString().padStart(2, '0');
        const minutes = now.getMinutes().toString().padStart(2, '0');
        // Форматируем для input type="datetime-local"
        const formattedNow = `${year}-${month}-${day}T${hours}:${minutes}`;
        startDateInput.value = formattedNow;

        // Можно установить endDate на будущее, например, через 1 день
        const defaultEndDate = new Date(now.getTime() + 24 * 60 * 60 * 1000);
        const endYear = defaultEndDate.getFullYear();
        const endMonth = (defaultEndDate.getMonth() + 1).toString().padStart(2, '0');
        const endDay = defaultEndDate.getDate().toString().padStart(2, '0');
        const endHours = defaultEndDate.getHours().toString().padStart(2, '0');
        const endMinutes = defaultEndDate.getMinutes().toString().padStart(2, '0');
        const formattedEnd = `${endYear}-${endMonth}-${endDay}T${endHours}:${endMinutes}`;
        endDateInput.value = formattedEnd;

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
        if (!data.start_date) {
            alert('Пожалуйста, укажите дату начала голосования.');
            return false;
        }
        if (!data.end_date) {
            alert('Пожалуйста, укажите дату окончания голосования.');
            return false;
        }

        const startDate = new Date(data.start_date);
        const endDate = new Date(data.end_date);

        if (endDate <= startDate) {
            alert('Дата окончания голосования должна быть позже даты начала.');
            return false;
        }
        if (data.min_votes <= 0) {
            alert('Минимальное количество голосов должно быть больше 0.');
            return false;
        }
        return true;
    }

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
            start_date: new Date(startDateInput.value).toISOString(), // Отправляем в ISO формате
            end_date: new Date(endDateInput.value).toISOString(), // Отправляем в ISO формате
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
                loadVotings(); // Reload votings on main page
                // If on profile page, update user data there too
                if (window.location.pathname === '/profile' && typeof window.fetchUserData === 'function') {
                    window.fetchUserData(userAddress);
                }
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

    // --- Main Votings List Logic ---
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

            votingCard.innerHTML = `
                <h3>${voting.title}</h3>
                <p>${voting.description}</p>
                <div class="voting-meta">
                    <span>Начало:<br>${new Date(voting.start_date).toLocaleString()}</span><br>
                    <span>Окончание:<br>${new Date(voting.end_date).toLocaleString()}</span>
                    <span class="${statusClass}">${statusText}</span>
                </div>
            `;
            votingsList.appendChild(votingCard);

            // Ensure the click listener is correctly attached
            votingCard.addEventListener('click', () => openVotingDetails(voting.voting_id));
        });
    }

    // --- Voting Details Modal Logic ---
    async function openVotingDetails(votingId) {
        currentVotingId = votingId; // Set currentVotingId when modal opens
        voteMessage.style.display = 'none';
        voteError.style.display = 'none';
        modalWinner.style.display = 'none';

        try {
            const response = await fetch(`/voting/${votingId}`);
            if (!response.ok) {
                throw new Error(`Failed to fetch voting details: ${response.statusText}`);
            }
            const voting = await response.json();

            modalVotingTitle.textContent = voting.title;
            modalVotingDescription.textContent = voting.description;
            modalCreatorAddress.textContent = voting.creator_address;
            modalStartDate.textContent = new Date(voting.start_date).toLocaleString();
            modalEndDate.textContent = new Date(voting.end_date).toLocaleString();
            modalVotesCount.textContent = voting.votes_count;
            modalMinVotes.textContent = voting.min_votes;

            modalStatus.textContent = voting.status;
            modalStatus.className = '';
            switch (voting.status) {
                case 'Upcoming':
                    modalStatus.classList.add('status-upcoming');
                    break;
                case 'Active':
                    modalStatus.classList.add('status-active');
                    break;
                case 'Finished':
                    modalStatus.classList.add('status-finished');
                    break;
                case 'Rejected':
                    modalStatus.classList.add('status-rejected');
                    break;
                default:
                    modalStatus.classList.add('status-unknown');
            }

            modalVotingOptions.innerHTML = '';
            voting.options.forEach((option, index) => {
                const optionDiv = document.createElement('div');
                optionDiv.className = 'vote-option-item';
                optionDiv.innerHTML = `
                    <input type="radio" name="voteOption" value="${index}" id="option${index}" ${voting.status !== 'Active' ? 'disabled' : ''}>
                    <label for="option${index}">${option.title} (${option.countVotes} голосов)</label>
                `;
                modalVotingOptions.appendChild(optionDiv);

                if (voting.status !== 'Active') {
                    optionDiv.classList.add('disabled');
                }

                optionDiv.addEventListener('click', () => {
                    if (voting.status === 'Active') {
                        document.querySelectorAll('.vote-option-item').forEach(item => item.classList.remove('selected'));
                        optionDiv.classList.add('selected');
                        optionDiv.querySelector('input[type="radio"]').checked = true;
                    }
                });
            });

            const userAddress = localStorage.getItem('userAddress');
            // Check if user has voted using the 'voters' map
            const hasUserVoted = userAddress && voting.voters && voting.voters[userAddress.toLowerCase()] && voting.voters[userAddress.toLowerCase()].is_voted;

            const canSubmit = voting.status === 'Active' && !hasUserVoted;
            submitVoteButton.disabled = !canSubmit;

            if (voting.status === 'Upcoming') {
                submitVoteButton.textContent = 'Голосование ещё не началось';
            } else if (voting.status === 'Finished' || voting.status === 'Rejected') {
                submitVoteButton.textContent = 'Голосование завершено';
            } else if (hasUserVoted) {
                submitVoteButton.textContent = 'Вы уже проголосовали';
            } else {
                submitVoteButton.textContent = 'Проголосовать';
            }

            // Display winner/rejected status
            if (voting.status === 'Finished') {
                modalWinner.textContent = `Победитель: ${voting.winner.join(', ')}`;
                modalWinner.style.display = 'block';
                modalWinner.className = 'modal-winner status-finished';
            } else if (voting.status === 'Rejected') {
                modalWinner.textContent = `Голосование отклонено: не набрано ${voting.min_votes} голосов.`;
                modalWinner.style.display = 'block';
                modalWinner.className = 'modal-winner status-rejected';
            } else {
                modalWinner.style.display = 'none';
            }

            votingDetailsModal.style.display = 'block';

        } catch (error) {
            console.error('Ошибка при загрузке деталей голосования:', error);
            alert('Не удалось загрузить детали голосования.');
            votingDetailsModal.style.display = 'none';
        }
    }

    // Event listeners for closing the details modal
    detailsCloseButton.addEventListener('click', () => {
        votingDetailsModal.style.display = 'none';
    });

    closeDetailsModalButton.addEventListener('click', () => {
        votingDetailsModal.style.display = 'none';
    });

    window.addEventListener('click', (event) => {
        if (event.target === createModal) {
            createModal.style.display = 'none';
        }
        if (event.target === votingDetailsModal) {
            votingDetailsModal.style.display = 'none';
        }
    });

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
        const votingId = currentVotingId; // Use the stored ID

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
                // Disable all radio buttons to prevent further votes in this session
                document.querySelectorAll('.vote-option-item input[type="radio"]').forEach(radio => radio.disabled = true);

                // Re-fetch and display details to update vote counts and status
                openVotingDetails(votingId);
                loadVotings(); // Refresh the main list

                if (window.location.pathname === '/profile') {
                    // Update user data on profile page if needed
                    if (typeof window.fetchUserData === 'function') {
                        window.fetchUserData(userAddress);
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

    // --- Initial Load Logic ---
    // This ensures votings are loaded and the UI is ready, regardless of MetaMask connection.
    loadVotings();
});