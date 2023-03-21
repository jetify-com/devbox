from django.shortcuts import render, redirect
from .models import Todo

def todo_list(request):
    todos = Todo.objects.all()
    context = {'todos': todos}
    return render(request, 'todo_app/todo_list.html', context)

def create_todo(request):
    if request.method == 'POST':
        title = request.POST.get('title')
        todo = Todo(title = title)
        todo.save()
        return redirect(todo_list)
    return render(request, 'todo_app/create_todo.html')
