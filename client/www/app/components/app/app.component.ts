import {Component} from 'angular2/core';
import {NodeListComponent} from '../nodelist/nodelist.component';
import {RouteConfig, ROUTER_DIRECTIVES} from 'angular2/router';
import {HTTP_PROVIDERS} from "angular2/http";
import {ROUTER_PROVIDERS} from "angular2/router";
import {ApiClient} from "../../services/apiclient";

@Component({
  selector: 'app',
  templateUrl: 'app/components/app/app.html',
  styleUrls: ['app/components/app/app.css'],
  directives: [NodeListComponent, ROUTER_DIRECTIVES],
  providers: [ApiClient, HTTP_PROVIDERS, ROUTER_PROVIDERS]
})
@RouteConfig([
  {
    path: '/browse/:parentId',
    name: 'Browse',
    component: NodeListComponent
  },
  {
    path: '/',
    redirectTo: ['Browse', {parentId: 'rootNode'}],
    useAsDefault: true
  }
])
export class AppComponent {}
