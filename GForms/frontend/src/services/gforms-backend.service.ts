import { inject, Injectable } from '@angular/core';
import {
  HttpClient,
  HttpHeaders,
  HttpErrorResponse,
} from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';

/**
 * Represents a single question within a form.
 */
export interface Question {
  description: any;
  required: any;
  placeholder: any;
  id: string; // UUID
  text: string;
  type: string; // Consider using a literal type if possible, e.g., 'rating' | 'text' | 'multiple_choice'
  extra_info: string;
  is_required: boolean;
  created_at: string | null; // ISO 8601 Date string
  updated_at: string | null; // ISO 8601 Date string
}

/**
 * Represents the main form structure.
 */
export interface Form {
  id: string; // UUID
  title: string;
  description: string;
  creator_user_id: string;
  questions: Question[]; // Array of Question objects
  created_at: string; // ISO 8601 Date string
  updated_at: string; // ISO 8601 Date string
}

/**
 * Represents the response object
 */
export interface FormResponse {
  id: string;
  form_id: string;
  respondent_user_id: string;
  answers: Answer[];
  created_at: string;
  updated_at: string;
}

export interface Answer {
  id: string;
  question_id: string;
  value: string;
  created_at: string;
  updated_at: string;
}

export type NewFormDto = Omit<
  Form,
  'id' | 'created_at' | 'updated_at' | 'questions'
> & { questions?: Omit<Question, 'id' | 'created_at' | 'updated_at'>[] };
export type UpdateFormDto = Partial<
  Omit<Form, 'id' | 'creator_user_id' | 'created_at' | 'updated_at'>
>;
export type NewFormResponse = Omit<
  FormResponse,
  'id' | 'created_at' | 'updated_at' | 'answers'
> & { answers?: Omit<Answer, 'id' | 'created_at' | 'updated_at'>[] };

@Injectable({
  providedIn: 'root',
})
// Renamed class to match the file name and purpose
export class FormService {
  // Adjusted apiUrl to point to a potential 'forms' endpoint. Modify if your backend uses a different path.
  private apiUrl = 'http://localhost:8080/forms';

  private http = inject(HttpClient);

  private httpOptions = {
    headers: new HttpHeaders({
      'Content-Type': 'application/json',
      // Add other headers here if needed (e.g., Authorization)
    }),
    withCredentials: true,
  };

  // --- Form Methods ---

  /**
   * GET: Retrieve a list of all forms.
   * @returns Observable<Form[]>
   */
  getForms(): Observable<Form[]> {
    return this.http
      .get<Form[]>(this.apiUrl, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  /**
   * GET: Retrieve a single form by its ID.
   * @param id The UUID string of the form.
   * @returns Observable<Form>
   */
  getFormById(id: string): Observable<Form> {
    const url = `${this.apiUrl}/${id}`;
    return this.http
      .get<Form>(url, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  /**
   * POST: Create a new form.
   * Assumes the backend handles ID and timestamp generation.
   * Questions might be included or added in a separate step depending on API design.
   * @param formData Data for the new form (title, description, creator_user_id, optional questions).
   * @returns Observable<Form> The newly created form object from the backend.
   */
  createForm(formData: NewFormDto): Observable<Form> {
    return this.http
      .post<Form>(this.apiUrl, formData, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  /**
   * PUT: Update an existing form.
   * Allows partial updates.
   * @param id The UUID string of the form to update.
   * @param formData An object containing the fields to update.
   * @returns Observable<Form> The updated form object from the backend.
   */
  updateForm(id: string, formData: UpdateFormDto): Observable<Form> {
    const url = `${this.apiUrl}/${id}`;
    // Using PUT here, some APIs might use PATCH for partial updates
    return this.http
      .put<Form>(url, formData, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  /**
   * DELETE: Delete a form by its ID.
   * @param id The UUID string of the form to delete.
   * @returns Observable<any> Typically returns no body on success (204 No Content).
   */
  deleteForm(id: string): Observable<any> {
    const url = `${this.apiUrl}/${id}`;
    return this.http
      .delete(url, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  // --- Responses Methods ---

  /**
   * GET: Retrieve all responses for a specific form.
   * @param formId The UUID string of the form.
   * @returns Observable<FormResponse[]> An array of response objects.
   */
  getFormResponses(formId: string): Observable<FormResponse[]> {
    const url = `${this.apiUrl}/${formId}/responses`;
    return this.http
      .get<FormResponse[]>(url)
      .pipe(catchError(this.handleError));
  }

  /**
   * @param formId
   * @param response
   * @returns
   */
  submitFormResponse(
    formId: string,
    response: NewFormResponse
  ): Observable<any> {
    const url = `${this.apiUrl}/${formId}/responses`;
    return this.http
      .post(url, response, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  updateQuestions(formId: string, questions: Question[]) {
    const url = `${this.apiUrl}/${formId}/questions`;
    return this.http
      .put(url, questions, this.httpOptions)
      .pipe(catchError(this.handleError));
  }

  // --- Error Handling ---
  private handleError(error: HttpErrorResponse) {
    let errorMessage = 'An unknown error occurred!';
    if (error.error instanceof ErrorEvent) {
      // Client-side or network error
      errorMessage = `Client-side error: ${error.error.message}`;
    } else {
      // Backend returned an unsuccessful response code
      errorMessage = `Server returned code ${error.status}, error message is: ${error.message}`;
      // You could try to extract a more specific error message from error.error if the backend sends one
      if (
        error.error &&
        typeof error.error === 'object' &&
        error.error.message
      ) {
        errorMessage += ` - ${error.error.message}`;
      } else if (error.error && typeof error.error === 'string') {
        errorMessage += ` - ${error.error}`;
      }
    }
    console.error(errorMessage, error); // Log the full error object too
    // Return an observable with a user-facing error message.
    return throwError(
      () =>
        new Error(
          'Something went wrong with the form operation. Please try again later.'
        )
    );
  }
}
