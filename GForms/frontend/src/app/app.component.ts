import { Component, CUSTOM_ELEMENTS_SCHEMA, OnInit } from '@angular/core';
// Import RouterLink along with RouterOutlet
import { CommonModule } from '@angular/common';
import { RouterLink, RouterOutlet } from '@angular/router';
import { Observable } from 'rxjs';
import { AccountService } from '../services/account-service.service';

@Component({
  selector: 'app-root',
  standalone: true,
  // Add RouterLink here:
  imports: [RouterLink, RouterOutlet, RouterLink, CommonModule],
  templateUrl: './app.component.html',
  styleUrl: './app.component.css',
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
})
export class AppComponent {
  title = 'gforms-app';
  isLoading: boolean = true;
  error: string | null = null;

  userLogged$: Observable<string | null>;

  constructor(private accountService: AccountService) {
    this.userLogged$ = this.accountService.currentUser$;
  }

  ngOnInit(): void {
    this.isLoading = false;
  }

  logout(): void {
    this.accountService.signout().subscribe({
      next: () => {
        console.log('Signout ok');
      },
      error: (err) => console.error('Signout Error: ', err)
    });
  }
}
