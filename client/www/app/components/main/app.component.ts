import {Component} from 'angular2/core';
import {FileListComponent} from '../filelist/filelist.component';
import {ApiClient} from '../../services/apiclient';
import {HTTP_PROVIDERS} from 'angular2/http';

@Component({
  selector: 'app',
  templateUrl: 'app/components/main/app.html',
  styleUrls: ['app/components/app/app.css'],
  directives: [FileListComponent],
  providers: [ApiClient, HTTP_PROVIDERS]
})
export class AppComponent {}
