package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var consumedTransfers []Transfer

type BankAccount struct {
	id             int
	balance        float64
	initialBalance float64
	mutex          sync.Mutex
	operations     []OperationRecord
}

type OperationRecord struct {
	id       int
	transfer Transfer
}

type Transfer struct {
	fromAccountId int
	toAccountId   int
	amount        float64
}

func invertTransfer(transfer Transfer) Transfer {
	return Transfer{transfer.toAccountId, transfer.fromAccountId, -transfer.amount}
}

func performTransfer(from, to *BankAccount, transfer Transfer) (bool, error) {
	// alawys lock the smaller id first to avoid deadlock
	first, second := from, to
	if from.id > to.id {
		first, second = to, from
	}
	// // lock the account with smaller id first
	first.mutex.Lock()
	second.mutex.Lock()

	// check if the from account has enough balance
	if from.balance < transfer.amount {
		// unlock the accounts
		second.mutex.Unlock()
		first.mutex.Unlock()
		return false, errors.New("insufficient balance")
	}

	// perfom the transfer
	from.balance -= transfer.amount
	to.balance += transfer.amount

	// record the operation
	from.recordOperation(invertTransfer(transfer))
	to.recordOperation(transfer)
	consumedTransfers = append(consumedTransfers, transfer)

	// unlock the accounts
	second.mutex.Unlock()
	first.mutex.Unlock()

	return true, nil
}

func (b *BankAccount) recordOperation(transfer Transfer) {
	// check if this is the first operation
	if len(b.operations) == 0 {
		b.operations = append(b.operations, OperationRecord{1, transfer})
	} else {
		// get the last operation id
		lastOperation := b.operations[len(b.operations)-1]
		// create the new operation record
		newOperation := OperationRecord{lastOperation.id + 1, transfer}
		// append the new operation to the operations list
		b.operations = append(b.operations, newOperation)
	}
}

func (b *BankAccount) consistencyCheck(consumedTransfers *[]Transfer) bool {
	// check if all the operations are consistent
	balance := b.initialBalance
	for _, operation := range b.operations {
		balance += operation.transfer.amount
	}

	// check if transfers made appear in the logs
	transfersMade := getTransfersContainingAccount(b, *consumedTransfers)
	for _, transfer := range transfersMade {
		found := false
		for _, operation := range b.operations {
			if operation.transfer == transfer || operation.transfer == invertTransfer(transfer) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return balance == b.balance
}

func checkAllAccountsConsistency(accounts []*BankAccount, consumedTransfers *[]Transfer) bool {
	// lock all the accounts
	for _, account := range accounts {
		account.mutex.Lock()
	}
	// check the consistency of all the accounts
	for _, account := range accounts {
		if !account.consistencyCheck(consumedTransfers) {
			// unlock all the accounts
			for _, account := range accounts {
				account.mutex.Unlock()
			}
			return false
		}
	}
	// unlock all the accounts
	for _, account := range accounts {
		account.mutex.Unlock()
	}
	return true
}

func startConsitencyCheck(accounts []*BankAccount, quitChan chan bool, consumedTransfers *[]Transfer) {
	for {
		select {
		case <-quitChan:
			return
		default:
			if !checkAllAccountsConsistency(accounts, consumedTransfers) {
				fmt.Println("Inconsistent state detected!")
			} else {
				fmt.Println("All accounts are consistent.")
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func getTransfersContainingAccount(account *BankAccount, transfers []Transfer) []Transfer {
	var transfersMade []Transfer
	for _, transfer := range transfers {
		if transfer.fromAccountId == account.id || transfer.toAccountId == account.id {
			transfersMade = append(transfersMade, transfer)
		}
	}
	return transfersMade
}

func main() {
	// create the accounts
	account1 := BankAccount{1, 1000, 1000, sync.Mutex{}, []OperationRecord{}}
	account2 := BankAccount{2, 1000, 1000, sync.Mutex{}, []OperationRecord{}}
	account3 := BankAccount{3, 1000, 1000, sync.Mutex{}, []OperationRecord{}}
	account4 := BankAccount{4, 1000, 1000, sync.Mutex{}, []OperationRecord{}}
	account5 := BankAccount{5, 1000, 1000, sync.Mutex{}, []OperationRecord{}}
	// create the accounts slice
	accounts := []*BankAccount{&account1, &account2, &account3, &account4, &account5}
	// create some transfers
	transfersToMake := []Transfer{
		{1, 2, 100},
		{2, 1, 50},
		{2, 3, 200},
		{3, 4, 300},
		{4, 5, 400},
		{5, 1, 100},
		{1, 3, 600},
		{3, 5, 700},
		{5, 2, 800},
		{2, 4, 50},
		{4, 1, 100},
		{1, 5, 20},
		{5, 3, 30},
		{3, 2, 20},
		{2, 5, 30},
		{5, 4, 40},
		{4, 3, 50},
		{1, 4, 10},
		{4, 2, 20},
		{2, 3, 30},
	}

	var wg sync.WaitGroup
	// start the consistency check
	quitChan := make(chan bool)
	go startConsitencyCheck(accounts, quitChan, &consumedTransfers)
	// perform the transfers
	for len(transfersToMake) > 0 {
		transfer := transfersToMake[0]
		transfersToMake = transfersToMake[1:]
		from := accounts[transfer.fromAccountId-1]
		to := accounts[transfer.toAccountId-1]

		wg.Add(1)
		go func(from *BankAccount, to *BankAccount, transfer Transfer) {
			defer wg.Done()
			_, err := performTransfer(from, to, transfer)
			if err != nil {
				fmt.Printf("Transfer from account %d to account %d failed: %s\n", from.id, to.id, err)
			} else {
				fmt.Printf("Transfer from account %d to account %d of amount %.2f completed successfully.\n", from.id, to.id, transfer.amount)
			}
		}(from, to, transfer)
		// wait before another transfer
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()
	// stop the consistency check
	quitChan <- true

	// check the consistency of all the accounts
	if !checkAllAccountsConsistency(accounts, &consumedTransfers) {
		fmt.Println("Inconsistent state detected!")
	} else {
		fmt.Println("All accounts are consistent.")
	}
	// print the balances of all the accounts
	for _, account := range accounts {
		fmt.Printf("Account ID: %d, Balance: %.2f\n", account.id, account.balance)
	}
}
