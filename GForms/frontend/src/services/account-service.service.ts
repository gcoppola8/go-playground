import { inject, Injectable, signal } from '@angular/core';
import {
  HttpClient,
  HttpHeaders,
  HttpErrorResponse,
} from '@angular/common/http';
import { BehaviorSubject, Observable, throwError } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';
import { Router } from '@angular/router';

// --- Interfaces (Adjust based on your actual API) ---

/**
 * Data Transfer Object for Sign-in request.
 */
export interface SigninDto {
  email: string;
  password: string;
}

/**
 * Data Transfer Object for Sign-up request.
 * Add other fields as needed (e.g., name, username).
 */
export interface SignupDto {
  username: string;
  email: string;
  password: string;
}

/**
 * Response object for authentication.
 * Adjust based on what your API returns (e.g., token, user info).
 */
export interface AuthResponse {
  username?: string;
  email?: string;
  message: string;
  error?: string;
}

@Injectable({
  providedIn: 'root',
})
export class AccountService {
  private apiUrl = 'http://localhost:8080/api/account'; // Base URL for account endpoints
  private http = inject(HttpClient);
  router = inject(Router);

  httpOptions = {
    headers: new HttpHeaders({
      'Content-Type': 'application/json',
      // Authorization headers might be added dynamically in methods or interceptors
    }),
    withCredentials: true,
  };

  private currentUserSource = new BehaviorSubject<string | null>(
    this.getInitialUser()
  );
  currentUser$ = this.currentUserSource.asObservable();

  constructor() {}

  /**
   * POST: Sign in a user.
   * @param credentials User's email and password.
   * @returns Observable<AuthResponse> containing token and user info.
   */
  signin(credentials: SigninDto): Observable<AuthResponse> {
    const url = `${this.apiUrl}/signin`;
    return this.http
      .post<AuthResponse>(url, credentials, this.httpOptions)
      .pipe(
        tap((response) => {
          if (response.error) {
            console.error('Login error:', response.error);
            this.signout();
          } else if (response.username) {
            this.currentUserSource.next(response.username);
          }
        }),
        catchError(this.handleError)
      );
  }

  /**
   * POST: Sign up a new user.
   * @param userData User's registration details.
   * @returns Observable<AuthResponse> containing token and user info (or adjust as needed).
   */
  signup(userData: SignupDto): Observable<AuthResponse> {
    const url = `${this.apiUrl}/signup`;
    return this.http
      .post<AuthResponse>(url, userData, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  /**
   * POST: Sign out the current user.
   * This might just clear local data or also call a backend endpoint.
   * Assuming a backend endpoint `/signout` exists for this example.
   * @returns Observable<void> indicating completion.
   */
  signout(): Observable<void> {
    // 1. Clear local storage/session storage where token might be stored
    // localStorage.removeItem('authToken'); // Example
    // sessionStorage.removeItem('authToken'); // Example

    // 2. Optionally call a backend endpoint to invalidate the session/token server-side
    const url = `${this.apiUrl}/signout`;
    // Adjust httpOptions if Authorization header is needed for signout
    return this.http
      .post<void>(url, {}, this.httpOptions) // Sending empty body
      .pipe(
        tap(() => {
          this.currentUserSource.next(null);
          this.router.navigate(['/account/login']);
        }),
        catchError(this.handleError)
      );
    // If signout is purely client-side (just clearing local token):
    // return new Observable<void>(observer => {
    //   try {
    //     // Clear local storage/session storage
    //     localStorage.removeItem('authToken'); // Example
    //     observer.next();
    //     observer.complete();
    //   } catch (error) {
    //     observer.error(error);
    //   }
    // });
    // Delete all cookies
  }

  /**
   * GET: Get the current user's information.
   * @returns Observable<AuthResponse> containing user information.
   */
  whoAmI(): Observable<AuthResponse> {
    const url = `${this.apiUrl}/whoami`;
    return this.http.get<AuthResponse>(url, this.httpOptions).pipe(
      tap((response) => {
        if (response.error) {
          console.error('WhoAmI error:', response.error);
          this.signout();
        }
      }),
      catchError(this.handleError)
    );
  }

  // --- Error Handling ---
  private handleError(error: HttpErrorResponse) {
    let errorMessage = 'An unknown error occurred during authentication!';
    if (error.error instanceof ErrorEvent) {
      // Client-side or network error
      errorMessage = `Client-side error: ${error.error.message}`;
    } else {
      // Backend returned an unsuccessful response code
      // Try to get message from backend response body
      const serverErrorMessage = error.error?.message || error.message;
      errorMessage = `Server returned code ${error.status}, error message is: ${serverErrorMessage}`;
    }
    console.error(errorMessage, error); // Log the full error object too
    // Return an observable with a user-facing error message.
    // Customize the user-facing message based on the status code if needed
    let userFriendlyMessage =
      'Authentication failed. Please check your details or try again later.';
    if (error.status === 401) {
      userFriendlyMessage =
        'Invalid credentials. Please check your email and password.';
    } else if (error.status === 409) {
      // Example: Conflict for signup if email exists
      userFriendlyMessage = 'This email address is already registered.';
    }
    return throwError(() => new Error(userFriendlyMessage));
  }
  // --- Helper to get initial user state (e.g., from storage) ---
  private getInitialUser(): string | null {
    const sessionCookie = document.cookie
      .split(';')
      .map((cookie) => cookie.trim())
      .find((cookie) => cookie.startsWith('gform_session='));

    if (sessionCookie) {
      this.whoAmI().subscribe({
        next: (response) => {
          if (response.username) {
            this.currentUserSource.next(response.username);
          }
        },
        error: (error) => {
          console.error('WhoAmI failed', error);
        },
      });
    }

    return null; // Default to not logged in
  }

  /** Gets the current value of the logged-in username */
  getCurrentUserValue(): string | null {
    return this.currentUserSource.value;
  }
}
