import {Injectable} from 'angular2/core';
import {Http, HTTP_PROVIDERS, Response} from 'angular2/http';
import {Observable} from 'rxjs/Observable';
import {NodeInfo} from '../models/nodeinfo';
import 'rxjs/Rx';

@Injectable()
export class ApiClient {

  constructor(private http: Http) {}

  listFiles(id: string): Observable<NodeInfo[]> {
    return this.http.get(`/v1/ls/${id}`)
      .map((res:Response) => res.json());
  }

  handleError(error: Response) {
    console.error(error);
  }
}
